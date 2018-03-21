# Walkthrough of MongoDB

This is an example of using Kanister to backup and restore MongoDB. In this example, we will deploy MongoDB with a sidecar container. This sidecar container will include the necessary tools to store protected data from MongoDB into an S3 bucket in AWS. Note that a sidecar container is not required to use Kanister, but rather is just one of several ways to access tools needed to protect the application. See other examples in the examples folder for alternative ways.

### 1. Deploy the Application

Deploy the example MongoDB application using the following command:
```bash
$ kubectl apply -f ./examples/mongo-sidecar/mongo-cluster.yaml
configmap "mongo-cluster" created
service "mongo-cluster" created
statefulset "mongo-cluster" created
```

Once MongoDB is running, you can populate it with some data. Let's add a collection called "restaurants" to a test database:
```bash
# Connect to MongoDB by running a shell inside Mongo's pod
$ kubectl exec -i -t mongo-cluster-0 -- bash -l

# From inside the shell, use the mongo CLI to insert some data into the test database
$ mongo test --quiet --eval "db.restaurants.insert({'name' : 'Roys', 'cuisine' : 'Hawaiian', 'id' : '8675309'})"
WriteResult({ "nInserted" : 1 })

# View the restaurants data in the test database
$ mongo test --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5a1dd0719dcbfd513fecf87c"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
```

### 2. Protect the Application

Next create a blueprint which describes how backup and restore actions can be executed on this application. The blueprint for this application can be found at `./examples/mongo-sidecar/blueprint.yaml`. Notice that the backup action of the blueprint references the S3 location specified in the config map in `./examples/mongo-sidecar/s3-location-configmap.yaml`. In order for this example to work, you should update the path field of s3-location-configmap.yaml to point to an S3 bucket to which you have access. You should also update secrets.yaml to contain the necessary AWS credentials to access this bucket. Provide your AWS credentials by setting the corresponding data values for `aws_access_key_id` and `aws_secret_access_key` in secrets.yaml. These are encoded using base64.

```bash
# Get base64 encoded aws keys
$ echo "YOUR_KEY" | base64

# Create the ConfigMap with an S3 path
$ kubectl apply -f ./examples/mongo-sidecar/s3-location-configmap.yaml
configmap "mongo-s3-location" created

# Create secrets containing the necessary AWS credentials
$ kubectl apply -f examples/mongo-sidecar/secrets.yaml

# Create the blueprint for MongoDB
$ kubectl apply -f ./examples/mongo-sidecar/blueprint.yaml
blueprint "mongo-sidecar" created
```

You can now take a backup of MongoDB's data using an action set defining backup for this application:
```bash
$ kubectl create -f ./examples/mongo-sidecar/backup-actionset.yaml
actionset "mongo-backup-12046" created

$ kubectl get actionsets.cr.kanister.io
NAME                KIND
mongo-backup12046   ActionSet.v1alpha1.cr.kanister.io
```

### 3. Disaster strikes!

Let's say someone with fat fingers accidentally deleted the restaurants collection using the following command:
```bash
# Drop the restaurants collection
$ mongo test --quiet --eval "db.restaurants.drop()"
true
```

If you try to access this data in the database, you should see that it is no longer there:
```bash
# No entries should be found in the restaurants collection
$ mongo test --quiet --eval "db.restaurants.find()"
$
```

### 4. Restore the Application

To restore the missing data, we want to use the backup created in step 2. An easy way to do this is to leverage kanctl, a command-line tool that helps create action sets that depend on other action sets:

```bash
$ kanctl perform restore --from "mongo-backup-12046"
actionset restore-mongo-backup-12046-s1wb7 created

# View the status of the actionset
$ kubectl get actionset restore-mongo-backup-12046-s1wb7 -oyaml
```

You should now see that the data has been successfully restored to MongoDB!
```bash
$ mongo test --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5a1dd0719dcbfd513fecf87c"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
```

### 5. Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl perform delete --from "mongo-backup-12046"
actionset "delete-mongo-backup-12046-kf8mt" created

# View the status of the actionset
$ kubectl get actionset delete-mongo-backup-12046-kf8mt -oyaml
```