# Walkthrough of MongoDB

This is an example of using Kanister to backup and restore MongoDB. In this example, we will deploy MongoDB with a sidecar container. This sidecar container will include the necessary tools to store protected data from MongoDB into an S3 bucket in AWS. Note that a sidecar container is not required to use Kanister, but rather is just one of several ways to access tools needed to protect the application. See other examples in the examples folder for alternative ways.


### Prerequisites

- Kubernetes 1.20+
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.110.0 installed in your cluster, let's assume in Namespace `kanister`
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)


### 1. Build and push mongo-sidecar docker image

The MongoDB application needs to be instrumented with the required tooling in order to get protected with Kanister. Follow the steps below to build a `mongo-sidecar` docker image which has all the necessary tools installed.

```bash
# Clone Kanister Github repo locally
$ git clone https://github.com/kanisterio/kanister.git <path_to_kanister>

# Build mongo-sidecar docker image
$ docker build -t <registry>/<repository>/mongo-sidecar:<tag_name> <path_to_kanister>/docker/kanister-mongodb-replicaset/image/
$ docker push <registry>/<repository>/mongo-sidecar:<tag_name>
```

### 2. Deploy the Application

Replace `<registry>`, `<repository>` and `<tag_name>` placeholders with actual values in `mongo-cluster.yaml`.

The following command deploys the example MongoDB application in `default` namespace:
```bash
$ kubectl apply -f ./examples/mongo-sidecar/mongo-cluster.yaml
configmap "mongo-cluster" created
service "mongo-cluster" created
statefulset "mongo-cluster" created
```

Once MongoDB is running, you can populate it with some data. Let's add a collection called "restaurants" to a test database:
```bash
# Connect to MongoDB by running a shell inside MongoDB's pod
$ kubectl exec -i -t mongo-cluster-0 -- bash -l

# From inside the shell, use the mongo CLI to insert some data into the test database
$ mongo test --quiet --eval "db.restaurants.insert({'name' : 'Roys', 'cuisine' : 'Hawaiian', 'id' : '8675309'})"
WriteResult({ "nInserted" : 1 })

# View the restaurants data in the test database
$ mongo test --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5a1dd0719dcbfd513fecf87c"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
```

### 3. Protect the Application

Next create a Blueprint which describes how backup and restore actions can be executed on this application. The Blueprint for this application can be found at `./examples/mongo-sidecar/blueprint.yaml`. Notice that the backup action of the Blueprint references the S3 location specified in the ConfigMap in `./examples/mongo-sidecar/s3-location-configmap.yaml`. In order for this example to work, you should update the path field of s3-location-configmap.yaml to point to an S3 bucket to which you have access. You should also update `secrets.yaml` to include AWS credentials that have read/write access to the S3 bucket. Provide your AWS credentials by setting the corresponding data values for `aws_access_key_id` and `aws_secret_access_key` in `secrets.yaml`. These are encoded using base64. The following commands will create a ConfigMap, Secrets and a Blueprint in controller's namespace:

```bash
# Get base64 encoded aws keys
$ echo -n "YOUR_KEY" | base64

# Create the ConfigMap with an S3 path
$ kubectl apply -f ./examples/mongo-sidecar/s3-location-configmap.yaml
configmap "mongo-s3-location" created

# Create the secrets with the AWS credentials
$ kubectl apply -f ./examples/mongo-sidecar/secrets.yaml
secrets "aws-creds" created

# Create the Blueprint for MongoDB
$ kubectl apply -f ./examples/mongo-sidecar/blueprint.yaml
blueprint "mongo-sidecar" created
```

You can now take a backup of MongoDB's data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.
```bash
$ kubectl --namespace kanister apply -f ./examples/mongo-sidecar/backup-actionset.yaml
actionset "mongo-backup-12046" created

$ kubectl --namespace kanister get actionsets.cr.kanister.io
NAME                KIND
mongo-backup-12046   ActionSet.v1alpha1.cr.kanister.io
```

### 4. Disaster strikes!

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

### 5. Restore the Application

To restore the missing data, we want to use the backup created in step 2. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
$ kanctl --namespace kanister create actionset --action restore --from "mongo-backup-12046"
actionset restore-mongo-backup-12046-s1wb7 created

# View the status of the ActionSet
kubectl --namespace kanister get actionset restore-mongo-backup-12046-s1wb7 -oyaml
```

You should now see that the data has been successfully restored to MongoDB!
```bash
$ mongo test --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5a1dd0719dcbfd513fecf87c"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
```

### 6. Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from "mongo-backup-12046"
actionset "delete-mongo-backup-12046-kf8mt" created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionset delete-mongo-backup-12046-kf8mt -oyaml
```
