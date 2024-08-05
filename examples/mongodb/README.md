# MongoDB

[MongoDB](https://www.mongodb.com/) is a cross-platform document-oriented database. Classified as a NoSQL database, MongoDB eschews the traditional table-based relational database structure in favor of JSON-like documents with dynamic schemas, making the integration of data in certain types of applications easier and faster.

## Prerequisites

* Kubernetes 1.20+
* Kubernetes beta APIs enabled only if `podDisruptionBudget` is enabled
* PV support on the underlying infrastructure
* Kanister controller version 0.110.0 installed in your cluster
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)

## Chart Details

We will be using bitnami [mongodb](https://github.com/bitnami/charts/tree/master/bitnami/mongodb) chart from official helm repo which bootstraps a [MongoDB](https://github.com/bitnami/bitnami-docker-mongodb) deployment on a [Kubernetes](http://kubernetes.io) cluster in replication mode using the [Helm](https://helm.sh) package manager.

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
$ helm repo add bitnami https://charts.bitnami.com/bitnami
$ helm repo update

$ helm install my-release bitnami/mongodb --namespace mongo-test --create-namespace \
	--set architecture="replicaset"
```

The command deploys MongoDB on the Kubernetes cluster in the mongo-test namespace


## Integrating with Kanister

If you have deployed mongodb application with other name than `my-release` and namespace other than `mongo-test`, you need to modify the commands used below to use the correct name and namespace

### Create Profile
Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--bucket <s3-bucket-name> --region <region-name> \
	--namespace mongo-test
```

**NOTE:**

The command will configure a location where artifacts resulting from Kanister data operations such as backup should go. This is stored as a `profiles.cr.kanister.io` CustomResource (CR) which is then referenced in Kanister ActionSets. Every ActionSet requires a Profile reference to complete the action. This CR (profiles.cr.kanister.io) can be shared between Kanister-enabled application instances.

**NOTE:**

If you have installed MongoDB chart using existing mongo secret by setting paramter `--set auth.existingSecret=<mongo-secret-name>` you will need to modify the secret name in the blueprint `mongo-blueprint.yaml` at following places:
```bash
actions.backup.phases[0].objects.mongosecret.name: <mongo-secret-name>
actions.restore.phases[0].objects.mongosecret.name: <mongo-secret-name>
```
### Create Blueprint

**NOTE: v2 Blueprints are experimental and are not supported with standalone Kanister.**

Create Blueprint in the same namespace as the controller

```bash
$ kubectl create -f ./mongo-blueprint.yaml -n kanister
```

Once MongoDB is running, you can populate it with some data. Let's add a collection called "restaurants" to a test database:

```bash
# Connect to MongoDB primary pod
$ kubectl exec -ti my-release-mongodb-0 -n mongo-test -- bash

# From inside the shell, use the mongo CLI to insert some data into the test database
$ mongosh admin --authenticationDatabase admin -u root -p $MONGODB_ROOT_PASSWORD --quiet --eval "db.restaurants.insertOne({'name' : 'Roys', 'cuisine' : 'Hawaiian', 'id' : '8675309'})"
{
  acknowledged: true,
  insertedId: ObjectId("6393065091e5c8cd94289f16")
}

# View the restaurants data in the test database
$ mongosh admin --authenticationDatabase admin -u root -p $MONGODB_ROOT_PASSWORD --quiet --eval "db.restaurants.find()"
[
  {
    _id: ObjectId("6393059b7844b3c1445e9d14"),
    name: 'Roys',
    cuisine: 'Hawaiian',
    id: '8675309'
  }
]
```


## Protect the Application

You can now take a backup of the MongoDB data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
$ kubectl get profile -n mongo-test
NAME               AGE
s3-profile-sph7s   2h

$ kanctl create actionset --action backup --namespace kanister --blueprint mongodb-blueprint --statefulset mongo-test/my-release-mongodb --profile mongo-test/s3-profile-sph7s
actionset backup-llfb8 created

# View the status of the actionset
$ kubectl --namespace kanister get actionsets.cr.kanister.io
NAME           PROGRESS   LAST TRANSITION TIME   STATE
backup-thkll   100.00     2022-12-09T10:01:42Z   complete
```

### Disaster strikes!

Let's say someone with fat fingers accidentally deleted the restaurants collection using the following command in mongodb primary pod:
```bash
# Connect to MongoDB primary pod
$ kubectl exec -ti my-release-mongodb-0 -n mongo-test -- bash

# Drop the restaurants collection
$ mongosh admin --authenticationDatabase admin -u root -p $MONGODB_ROOT_PASSWORD --quiet --eval "db.restaurants.drop()"
true
```

If you try to access this data in the database, you should see that it is no longer there:
```bash
$ mongosh admin --authenticationDatabase admin -u root -p $MONGODB_ROOT_PASSWORD --quiet --eval "db.restaurants.find()"
# No entries should be found in the restaurants collection
```

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

**NOTE:**

As a part of restore operation in MongoDB ReplicaSet, we are deleting data from the Secondary replicas to allow MongoDB to use `Initial Sync` for updating Secondaries as documented [here](https://docs.mongodb.com/manual/tutorial/restore-replica-set-from-backup/#update-secondaries-using-initial-sync)

```bash
$ kanctl --namespace kanister create actionset --action restore --from "backup-llfb8"
actionset restore-backup-llfb8-64gqm created

# View the status of the ActionSet
kubectl --namespace kanister get actionset restore-backup-llfb8-64gqm
NAME                         PROGRESS   LAST TRANSITION TIME   STATE
restore-backup-llfb8-64gqm   100.00     2022-12-09T10:06:39Z   complete
```

You should now see that the data has been successfully restored to MongoDB!

```bash
# Connect to MongoDB primary pod
$ kubectl exec -ti my-release-mongodb-0 -n mongo-test -- bash

$ mongosh admin --authenticationDatabase admin -u root -p $MONGODB_ROOT_PASSWORD --quiet --eval "db.restaurants.find()"
[
  {
    _id: ObjectId("6393059b7844b3c1445e9d14"),
    name: 'Roys',
    cuisine: 'Hawaiian',
    id: '8675309'
  }
]
```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from "backup-llfb8" --namespacetargets kanister
actionset "delete-backup-llfb8-k9ncm" created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionset delete-backup-llfb8-k9ncm
NAME                         PROGRESS   LAST TRANSITION TIME   STATE
delete-backup-llfb8-k9ncm    100.00     2022-12-09T10:08:20Z   complete
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-llfb8-64gqm -n kanister
```

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release -n mongo-test
```

The command removes all the Kubernetes components associated with the chart and deletes the release.
To completely remove the release include the `--purge` flag.

Delete Blueprint, Profile CR and ActionSet

```bash
$ kubectl delete blueprints.cr.kanister.io mongodb-blueprint -n kanister

$ kubectl get profiles.cr.kanister.io -n mongo-test
NAME               AGE
s3-profile-sph7s   2h

$ kubectl delete profiles.cr.kanister.io s3-profile-sph7s -n mongo-test

$ kubectl delete actionset backup-llfb8 restore-backup-llfb8-64gqm delete-backup-llfb8-k9ncm -n kanister
```
