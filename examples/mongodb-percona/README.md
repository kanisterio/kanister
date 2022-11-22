# MongoDB

[MongoDB](https://www.mongodb.com/) is a cross-platform document-oriented database. Classified as a NoSQL database, MongoDB eschews the traditional table-based relational database structure in favor of JSON-like documents with dynamic schemas, making the integration of data in certain types of applications easier and faster.

## Prerequisites

* Kubernetes 1.9+
* PV support on the underlying infrastructure
* Kanister controller version 0.84.0 installed in your cluster
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## Operator Details

We will be using [percona operator for Mongodb](https://docs.percona.com/percona-operator-for-mongodb/index.html), the blueprint follow backup and restore workflow as described in the [backup/restore documentation](https://docs.percona.com/percona-operator-for-mongodb/backups.html) of the operator. 

## Limitations 

For simplicity we did not patch the PerconaServerMongoDB object with the profiles.cr.kanister.io to define the backup target, you have to define it and the secret reference a secret that must define the AWS_ACCESS_KEY_ID and in the AWS_SECRET_ACCESS_KEY keys: 
```
    storages:
      my-s3-storage:
        type: s3
        s3:
          bucket: my-bucket
          credentialsSecret: s3-secret
          region: eu-west-3
          prefix: "mongodb/my-cluster-name/"
          uploadPartSize: 10485760
          maxUploadParts: 10000
          storageClass: STANDARD
          insecureSkipTLSVerify: false
```

Those values need to be defined (this section and the s3-secret) before you execute the blueprint.

## Install the operator and create a cluster

Edit cr.yaml and ensure the backup/storages section (line 443) is consistent with your s3 target, you may change : 
- the region
- the bucket name
- the endpoint
- the prefix 
- ...

Also define the env variable `AWS_S3_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` according to your setting.

Those values can't be obtained from a kanister profile because they need to be defined in the PerconaServerMongoDB object itself before any kanister action (see Limitations)

```
kubectl create namespace mongodb
kubectl config set-context --current --namespace=mongodb
kubectl apply -f https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/v1.13.0/deploy/bundle.yaml
kubectl create secret generic s3-secret --from-literal AWS_ACCESS_KEY_ID=$AWS_S3_ACCESS_KEY_ID --from-literal AWS_SECRET_ACCESS_KEY=$AWS_S3_SECRET_ACCESS_KEY
kubectl apply -f cr.yaml
```

check the status of the mongodb cluster
```
kubectl get psmdb
```

**NOTE:**

We also enable point in time restore (pitr) for this cluster

```
    pitr:
      enabled: true
      compressionType: gzip
      compressionLevel: 6
```

## Integrating with Kanister

If you have deployed the operator with other name than `my-cluster-name` and namespace other than `mongodb`, you need to modify the commands used below to use the correct name and namespace

### Create Profile
Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--bucket <s3-bucket-name> --region <region-name> \
	--namespace mongodb
```

**NOTE:**

The command will configure a location where artifacts resulting from Kanister data operations such as backup should go. This is stored as a profiles.cr.kanister.io CustomResource (CR) which is then referenced in Kanister ActionSets. Every ActionSet requires a Profile reference to complete the action. This CR (profiles.cr.kanister.io) can be shared between Kanister-enabled application instances.

**NOTE:**

The profile is actually useless in this case but mandatory for executing kanister action.


### Create Blueprint
Create Blueprint in the same namespace as the controller

```bash
$ kubectl create -f psmdb-bp.yaml -n kasten-io
```

Once MongoDB is running, you can populate it with some data. Let's add a collection called "ticker":

```bash
MONGODB_DATABASE_ADMIN_PASSWORD=$(kubectl get secret my-cluster-name-secrets -ojsonpath='{.data.MONGODB_DATABASE_ADMIN_PASSWORD}'|base64 -d)
MONGODB_DATABASE_ADMIN_USER=$(kubectl get secret my-cluster-name-secrets -ojsonpath='{.data.MONGODB_DATABASE_ADMIN_USER}'|base64 -d)
kubectl run -i --rm --tty percona-client \
    --image=percona/percona-server-mongodb:4.4.16-16 \
    --env=MONGODB_DATABASE_ADMIN_PASSWORD=$MONGODB_DATABASE_ADMIN_PASSWORD \
    --env=MONGODB_DATABASE_ADMIN_USER=$MONGODB_DATABASE_ADMIN_USER \
    --restart=Never \
    -- bash -il
mongo "mongodb://$MONGODB_DATABASE_ADMIN_USER:$MONGODB_DATABASE_ADMIN_PASSWORD@my-cluster-name-mongos.mongodb.svc.cluster.local/admin?ssl=false"
```

Insert some data 
```
db.createCollection("ticker")
db.ticker.insert(  {createdAt: new Date(), randomdata: "qstygshgqsfxxtqsfgqfhjqhsj"} )
db.ticker.insert(  {createdAt: new Date(), randomdata: "qstygshgqsfxxtqsfgqfhjqhsj"} )
db.ticker.insert(  {createdAt: new Date(), randomdata: "qstygshgqsfxxtqsfgqfhjqhsj"} )
db.ticker.insert(  {createdAt: new Date(), randomdata: "qstygshgqsfxxtqsfgqfhjqhsj"} )
db.ticker.find({}).sort({createdAt:-1}).limit(1)
```

In order to test Point In Time Restore (PITR) create a ticker pod that add new entry every seconds.

```
MONGODB_DATABASE_ADMIN_PASSWORD=$(kubectl get secret my-cluster-name-secrets -ojsonpath='{.data.MONGODB_DATABASE_ADMIN_PASSWORD}'|base64 -d)
MONGODB_DATABASE_ADMIN_USER=$(kubectl get secret my-cluster-name-secrets -ojsonpath='{.data.MONGODB_DATABASE_ADMIN_USER}'|base64 -d)
kubectl run percona-ticker \
    --image=percona/percona-server-mongodb:4.4.16-16 \
    --env=MONGODB_DATABASE_ADMIN_PASSWORD=$MONGODB_DATABASE_ADMIN_PASSWORD \
    --env=MONGODB_DATABASE_ADMIN_USER=$MONGODB_DATABASE_ADMIN_USER \
    -- bash -c "while true; do mongo \"mongodb://$MONGODB_DATABASE_ADMIN_USER:$MONGODB_DATABASE_ADMIN_PASSWORD@my-cluster-name-mongos.mongodb.svc.cluster.local/admin?ssl=false\" --eval 'db.ticker.insert(  {createdAt: new Date(), randomdata: \"qstygshgqsfxxtqsfgqfhjqhsj\"} )'; sleep 1; done"
```

Reuse the percona-client to check you have a new entry every second, execute this command multiple times.
```
db.ticker.find({}).sort({createdAt:-1}).limit(1)
```

**NOTE:**

If you change the storage name for something else than `my-s3-storage` in the cr.yaml file you need to change this name in the blueprint accordingly.

## Protect the Application

You can now take a backup of the MongoDB data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
$ kubectl get profiles.cr.kanister.io  -n mongodb
NAME               AGE
s3-profile-bvc8k   2h

$ kanctl create actionset \
   --action backup --namespace kasten-io \
   --blueprint psmdb-bp \
   --profile mongodb/s3-profile-bvc8k \
   --objects psmdb.percona.com/v1/perconaservermongodbs/mongodb/my-cluster-name

$ kubectl --namespace kasten-io get actionsets.cr.kanister.io
NAME                 AGE
backup-vvskw         2h

# View the status of the actionset
$ kubectl --namespace kasten-io describe actionset backup-vvskw
```

### Disaster strikes!

Let's say someone with fat fingers accidentally deleted the mongo cluster and all the pvcs:
```bash
kubectl delete psmdb my-cluster-name
kubectl delete po percona-ticker
kubectl delete pvc --all
```

All pods and pvc are now gone. Recreate the cluster 
```
kubectl apply -f cr.yaml
```

Check the cluster is now ready 
```
kubectl get psmdb
```

If you try to access this data in the database, you should see that it is no longer there:
```bash
MONGODB_DATABASE_ADMIN_PASSWORD=$(kubectl get secret my-cluster-name-secrets -ojsonpath='{.data.MONGODB_DATABASE_ADMIN_PASSWORD}'|base64 -d)
MONGODB_DATABASE_ADMIN_USER=$(kubectl get secret my-cluster-name-secrets -ojsonpath='{.data.MONGODB_DATABASE_ADMIN_USER}'|base64 -d)
kubectl run -i --rm --tty percona-client \
    --image=percona/percona-server-mongodb:4.4.16-16 \
    --env=MONGODB_DATABASE_ADMIN_PASSWORD=$MONGODB_DATABASE_ADMIN_PASSWORD \
    --env=MONGODB_DATABASE_ADMIN_USER=$MONGODB_DATABASE_ADMIN_USER \
    --restart=Never \
    -- bash -il
mongo "mongodb://$MONGODB_DATABASE_ADMIN_USER:$MONGODB_DATABASE_ADMIN_PASSWORD@my-cluster-name-mongos.mongodb.svc.cluster.local/admin?ssl=false"
```

You should have no output when trying to query the ticker collections
```
db.ticker.find({}).sort({createdAt:-1}).limit(1)
```


### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:


```bash
$ kanctl --namespace kasten-io create actionset --action restore --from "backup-vvskw"
actionset restore-backup-vvskw-sfpm6 created

# View the status of the ActionSet
kubectl --namespace kasten-io describe actionset restore-backup-vvskw-sfpm6
```

You should now see that the data has been successfully restored to MongoDB!

```bash
db.ticker.find({}).sort({createdAt:-1}).limit(1)
{ "_id" : ObjectId("637372caf98744ac5a6e4aae"), "createdAt" : ISODate("2022-11-15T11:06:50.296Z"), "randomdata" : "qstygshgqsfxxtqsfgqfhjqhsj" }
```

#### Point In Time Restore 

Let's say the backup was at 15th Nov 2022 at 15:11 and we want to restore at a specific point in time after the backup at 15:13:10 the same day

```
kanctl --namespace kasten-io create actionset --action restore --from "backup-vvskw" --options pitr="2022-11-15 15:13:10"
```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kasten-io create actionset --action delete --from "backup-vvskw"
actionset delete-backup-vvskw-xh29t created

# View the status of the ActionSet
$ kubectl --namespace kasten-io describe actionset delete-backup-vvskw-xh29t
```

**NOTE:**

To have the delete action working  we need the operator up and running.

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kasten-io logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-vvskw-sfpm6 -n kasten-io
```

## Uninstalling the operator and the blueprint

To uninstall/delete the mongodb cluster and the operator:

```bash
kubectl delete psmdb my-cluster-name
kubectl delete po percona-ticker
kubectl delete pvc --all
kubectl delete -f https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/v1.13.0/deploy/bundle.yaml
kubectl delete ns mongodb
```

The command removes all the Kubernetes components associated with the operator and deletes the mongodb namespace.

Delete Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io psmdb-bp -n kasten-io

$ kubectl get profiles.cr.kanister.io -n mongodb
NAME               AGE
s3-profile-bvc8k   2h          

$ kubectl delete profiles.cr.kanister.io s3-profile-bvc8k -n mongodb
```
