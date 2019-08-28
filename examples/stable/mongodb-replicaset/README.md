# MongoDB Helm Chart

## Prerequisites Details

* Kubernetes 1.9+
* Kubernetes beta APIs enabled only if `podDisruptionBudget` is enabled
* PV support on the underlying infrastructure
* Kanister version 0.20.0 with `profiles.cr.kanister.io` CRD installed
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## StatefulSet Details

* https://kubernetes.io/docs/concepts/abstractions/controllers/statefulsets/

## StatefulSet Caveats

* https://kubernetes.io/docs/concepts/abstractions/controllers/statefulsets/#limitations

## Chart Details

This chart implements a dynamically scalable [MongoDB replica set](https://docs.mongodb.com/manual/tutorial/deploy-replica-set/)
using Kubernetes StatefulSets and Init Containers.

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
$ helm repo add stable https://kubernetes-charts.storage.googleapis.com/
$ helm repo update

$ helm install --name my-release stable/mongodb-replicaset --namespace mongo-test
```

The command deploys MongoDB ReplicaSet on the Kubernetes cluster in the default
configuration. The [configuration](#configuration) section lists the parameters that can be
configured during installation.


## Integrating with Kanister

If you have deployed mongodb application with other name than `my-release` and namespace other than `mongo-test`, you need to modify the commands used below to use the correct name and namespace

### Create Profile
Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key 'AKIAIOSFODNN7EXAMPLE' \
	--secret-key 'wJalrXUtnFEMI%K7MDENG%bPxRfiCYEXAMPLEKEY' \
	--bucket <s3-bucket-name> --region ap-south-1 \
	--namespace mongo-test
```

**NOTE:**

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference whether one created as part of the application install or
not. Support for creating an ActionSet as part of install is simply for convenience.
This CR can be shared between Kanister-enabled application instances so one option is to
only create as part of the first instance.


### Create Blueprint
Create Blueprint in the same namespace as the controller

```bash
$ kubectl create -f kanister/mongodb-replicaset-blueprint.yaml -n kasten-io
```

Once MongoDB is running, you can populate it with some data. Let's add a collection called "restaurants" to a test database:

```bash
# Connect to MongoDB by running a shell inside MongoDB's pod
$ kubectl exec --namespace mongo-test -i -t my-release-mongodb-replicaset-0  -- bash -l

# From inside the shell, use the mongo CLI to insert some data into the test database
$ mongo test --quiet --eval "db.restaurants.insert({'name' : 'Roys', 'cuisine' : 'Hawaiian', 'id' : '8675309'})"
WriteResult({ "nInserted" : 1 })

# View the restaurants data in the test database
$ mongo test --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5a1dd0719dcbfd513fecf87c"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
```


## Protect the Application

You can now take a backup of the MongoDB data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
$ kubectl get profile -n mongo-test
NAME               AGE
s3-profile-sph7s   2h

$ kanctl create actionset --action backup --namespace kasten-io --blueprint mongodb-replicaset-blueprint --statefulset mongo-test/my-release-mongodb-replicaset --profile mongo-test/s3-profile-sph7s
actionset backup-llfb8 created

$ kubectl --namespace kasten-io get actionsets.cr.kanister.io
NAME                 AGE
backup-llfb8         2h

# View the status of the actionset
$ kubectl --namespace kasten-io describe actionset backup-llfb8
```

### Disaster strikes!

Let's say someone with fat fingers accidentally deleted the restaurants collection using the following command:
```bash
# Drop the restaurants collection
$ mongo test --quiet --eval "db.restaurants.drop()"
true
```

If you try to access this data in the database, you should see that it is no longer there:
```bash
$ mongo test --quiet --eval "db.restaurants.find()"
# No entries should be found in the restaurants collection
```

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
$ kanctl --namespace kasten-io create actionset --action restore --from "backup-llfb8"
actionset restore-backup-llfb8-64gqm created

# View the status of the ActionSet
kubectl --namespace kasten-io describe actionset restore-backup-llfb8-64gqm
```

You should now see that the data has been successfully restored to MongoDB!

```bash
$ mongo test --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5a1dd0719dcbfd513fecf87c"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kasten-io create actionset --action delete --from "backup-llfb8"
actionset "delete-backup-llfb8-k9ncm" created

# View the status of the ActionSet
$ kubectl --namespace kasten-io describe actionset delete-backup-llfb8-k9ncm
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kasten-io logs -l app=kanister-operator
```

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.
To completely remove the release include the `--purge` flag.

Delete Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io mongodb-replicaset-blueprint -n kasten-io

$ kubectl get profiles.cr.kanister.io -n mongo-test
NAME               AGE
s3-profile-sph7s   2h
$ kubectl delete profiles.cr.kanister.io s3-profile-sph7s -n mongo-test
```
