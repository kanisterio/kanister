This example shows how you can backup and restore the application that is deployed using [DeploymentConfig](https://docs.OpenShift.com/container-platform/4.1/applications/deployments/what-deployments-are.html#deployments-and-deploymentconfigs_what-deployments-are),
[OpenShift](https://www.openshift.com/)'s resource on an OpenShift cluster. Like Deployment, Statefulset in Kubernetes, OpenShift has another controller named DeploymentConfig, it is almost like Deployment but has some significant differences.

# DeploymentConfig

[DeploymentConfig](https://docs.openshift.com/container-platform/4.1/applications/deployments/what-deployments-are.html#deployments-and-deploymentconfigs_what-deployments-are) is not standard
Kubernetes resource but [OpenShift](https://www.openshift.com/) resource and creates a new
ReplicationController and let's it start up Pods.

This example can be followed if your application is deployed on [OpenShift](https://www.openshift.com/)
cluster's DeploymentConfig resources.

## Prerequisites

- Setup OpenShift, you can follow steps mentioned below
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.110.0 installed in your cluster in namespace `kanister`
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

**Note**

All the helm commands in the document are for the Helm version 3, if you have version 2 setup you will have to change the Helm commands a bit.

# Setup OpenShift

To test the applications that are deployed on OpenShift using DeploymentConfig, we will first have to setup
OpenShift cluster. To setup the OpenShift cluster in our local environment we are going to use [minishift](https://github.com/minishift/minishift).
To install and setup minishift please follow [this guide](https://docs.okd.io/latest/minishift/getting-started/index.html).

Once we have minishift setup we will go ahead and deploy MongoDB application on the minishift cluster using DeploymentConfig and try to backup and restore the data from the application.

**Note**

Once you have setup minishift by following the steps mentioned above, you can interact with the cluster using `oc` command line tool. By default
you are logged in as developer user, that will prevent us from creating some of the resources so please make sure you login as admin by
the following command

```bash
oc login -u system:admin
```
and you are all set to go through rest of this example.


# Install MongoDB

We will use the JSON templates, as described [here](https://github.com/openshift/origin/tree/master/examples/db-templates), provided by OpenShift to deploy the MongoDB database on the minishift cluster that we have just setup.

To install the MongoDB on your cluster please run below command

```bash
# Create the namespaces
~ oc create ns mongodb-test
namespace/mongodb-test created

# Install the database
~ oc new-app https://raw.githubusercontent.com/openshift/origin/master/examples/db-templates/mongodb-persistent-template.json \
  -p MONGODB_ADMIN_PASSWORD=secretpassword -n mongodb-test
```

we can use the parameter `MONGODB_ADMIN_PASSWORD` to setup the admin password for the MongoDB that we are installing. Once the database is installed get all the pods from `mongodb-test` namespace to make sure the pods are in `Running` status.

```bash
~  oc get pods -n mongodb-test
NAME              READY     STATUS    RESTARTS   AGE
mongodb-1-7vw4g   1/1       Running   0          1m
```

## Integrating with Kanister

When we say integrating with Kanister, we actually mean creating some Kanister resources to support `backup` and `restore`
actions.

### Create Profile

Create Profile Kanister resource using below command

```bash
~ kanctl create profile s3compliant --access-key <ACCESS-KEY> \
        --secret-key <SECRET-KEY> \
        --bucket <BUKET-NAME> --region <AWS-REGION> \
        --namespace mongodb-test
secret 's3-secret-gkvgi4' created
profile 's3-profile-vzpfb' created
```

**NOTE:**

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.


**NOTE:**

If MongoDB chart is installed specifying existing secret by setting parameter `--set
auth.existingSecret=<mongo-secret-name>` you will need to modify the secret name in
the blueprint `mongo-blueprint.yaml` at following places:

```bash
actions.backup.phases[0].objects.mongosecret.name: <mongo-secret-name>
actions.restore.phases[0].objects.mongosecret.name: <mongo-secret-name>
```

### Create Blueprint

**NOTE: v2 Blueprints are experimental and are not supported with standalone Kanister.**

Create Blueprint in the same namespace as the controller (`kanister`)

```bash
~ oc create -f mongo-dep-config-blueprint.yaml -n kanister
blueprint.cr.kanister.io/mongodb-blueprint created
```

Now that we have created the Profile and Blueprint Kanister resources we will insert some data into
MongoDB database that we will take backup of.
To insert the data into the MongoDB database we will `exec` into the MongoDB pod, please follow below commands
to do so

```bash
# Get the MongoDB pod name
~ oc get pods -n mongodb-test
NAME              READY     STATUS    RESTARTS   AGE
mongodb-1-7vw4g   1/1       Running   0          4m

# Exec into the pod and insert some records
~  oc exec -it -n mongodb-test mongodb-1-7vw4g bash
bash-4.2$ mongo admin --authenticationDatabase admin -u admin -p $MONGODB_ADMIN_PASSWORD --quiet --eval "db.restaurants.insert({'name' : 'Roys', 'cuisine' : 'Hawaiian', 'id' : '8675309'})"
WriteResult({ "nInserted" : 1 })

bash-4.2$ mongo admin --authenticationDatabase admin -u admin -p $MONGODB_ADMIN_PASSWORD --quiet --eval "db.restaurants.insert({'name' : 'Willies', 'cuisine' : 'Hawaiian', 'id' : '8675310'})"
WriteResult({ "nInserted" : 1 })

# view the record that you have just inserted
bash-4.2$ mongo admin --authenticationDatabase admin -u admin -p $MONGODB_ADMIN_PASSWORD --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5e5ce013d026502dd164d659"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
{ "_id" : ObjectId("5e5ce01966f876c088fb6697"), "name" : "Willies", "cuisine" : "Hawaiian", "id" : "8675310" }

```
As you can see we have inserted two documents in the `restaurants` collection.

## Protect the Application

You can now take a backup of the MongoDB data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
# Get the profile
~ oc get profile -n mongodb-test
NAME               AGE
s3-profile-vzpfb   1m

# Create the backup Actionset
~ kanctl create actionset --action backup --namespace kanister --blueprint mongodb-blueprint --deploymentconfig mongodb-test/mongodb \
    --profile mongodb-test/s3-profile-vzpfb
actionset backup-hdrxr created

# You can describe the Actionset to make sure it is completed
~ oc describe actionset -n kanister backup-hdrxr
# and you will be able to see the status in events at the very end, somewhat like below
# Normal  Update Complete  1m    Kanister Controller  Updated ActionSet 'backup-hdrxr' Status->complete
```

### Disaster strikes!

Let's say someone accidentally deleted the `restaurants` collection using the following command in mongodb pod:

```bash
# login to the MongoDB pod and delete the records manually
bash-4.2$ mongo admin --authenticationDatabase admin -u admin -p $MONGODB_ADMIN_PASSWORD --quiet --eval "db.restaurants.drop()"
true
```
If you try to access this data in the database, you should see that it is no longer there:

```bash
bash-4.2$ mongo admin --authenticationDatabase admin -u admin -p $MONGODB_ADMIN_PASSWORD --quiet --eval "db.restaurants.find()"
# No entries should be found in the database
```

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
# Get the backup Actionset name
~ oc get actionset -n kanister
NAME           AGE
backup-hdrxr   9m

# create the restore Actionset
~  kanctl --namespace kanister create actionset --action restore --from "backup-hdrxr"
actionset restore-backup-hdrxr-tdw9f created

# Describe the Actionset to check if it has been completed
~ oc describe actionset -n kanister restore-backup-hdrxr-tdw9f

# And you should see the last line of the describe, like below
# Normal  Update Complete  35s   Kanister Controller  Updated ActionSet 'restore-backup-hdrxr-tdw9f' Status->complete
```

Once you have verified that the status of Actionset is `Complete`, let's login to the MongoDB pod once again to make sure the data has been restored.

```bash
# Exec into the MongoDB pod
bash-4.2$ mongo admin --authenticationDatabase admin -u admin -p $MONGODB_ADMIN_PASSWORD --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5e5ce013d026502dd164d659"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
{ "_id" : ObjectId("5e5ce01966f876c088fb6697"), "name" : "Willies", "cuisine" : "Hawaiian", "id" : "8675310" }
```

As you can see the data that we deleted in previous step, to imitate disaster, has been restored successfully.

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
~ kanctl --namespace kanister create actionset --action delete --from "backup-hdrxr"
actionset delete-backup-hdrxr-fbllx created
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset <actionset-name> -n kanister
```

## Uninstalling the database

When we install a application using `oc new-app` a label of format `app=<name>`, gets added to all the resources that get created. To get that label execute below command

```bash
~ oc get pods -n mongodb-test  --show-labels
NAME              READY     STATUS    RESTARTS   AGE       LABELS
mongodb-1-7vw4g   1/1       Running   0          27m       app=mongodb-persistent,deployment=mongodb-1,deploymentconfig=mongodb,name=mongodb
```
As you can see the label that was added is `app=mongodb-persistent` to delete all the resources with this label, use below command

```bash
~  oc delete all -n mongodb-test -l app=mongodb-persistent
```

### Delete Kanister resources
Remove Blueprint and Profile CR

```bash
~ oc delete blueprints.cr.kanister.io <blueprint-name> -n kanister

~ oc delete profiles.cr.kanister.io <profile-name> -n mongodb-test
```
