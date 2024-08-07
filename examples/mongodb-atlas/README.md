# MongoDB Atlas

[MongoDB Atlas](https://www.mongodb.com/atlas) is an integrated suite of cloud
database and data services to accelerate and simplify how you build with data.
It deploys and scales a MongoDB cluster in the cloud.

## Prerequisites

* Kubernetes 1.20+
* Kanister controller version 0.110.0 installed in your cluster
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)
* Already provisioned MongoDB Atlas cluster (https://www.mongodb.com/docs/atlas/getting-started)

## Integrating with Kanister

In case, if you don't have `Kanister` installed already, use following commands
to install.

```bash
$ helm repo add kanister https://charts.kanister.io
$ helm install kanister --namespace kanister --create-namespace \
    kanister/kanister-operator --set image.tag=0.110.0
```

### Create Blueprint

Create Blueprint in the same namespace as the Kanister controller.

```bash
$ kubectl create -f ./mongodb-atlas-blueprint.yaml -n kanister
```

### Create Secret to store MongoDB Atlas Account details

Please create a secret to store the MongoDB Atlas account details that are
going to get used by `atlas` command in the blueprint.

`Public_Key`: Public key included in Organization level API Key
(you'll find it in Organizations > Access Manager > API Keys)

`Private_Key`: Private key included in Organization level API Key
(you'll find it in Organizations > Access Manager > API Keys)

`Org_Id`: A unique 24 characters Organization ID
(you'll find it in Organizations > Settings)

`Project_Id`: A unique 24 character Project ID
(you'll find it in Organizations > Project > Settings)

`Cluster_Name`: Cluster name in project
(you'll find it in Organizations > Project > Database)

```bash
$ kubectl create namespace mongodb-atlas-test
namespace/mongodb-atlas-test created

$ kubectl create secret generic mongo-atlas-secret \
    --from-literal=publickey="<Public_Key>" \
    --from-literal=privatekey="<Private_Key>" \
    --from-literal=orgid="<Org_Id>" \
    --from-literal=projectid="<Project_Id>" \
    --from-literal=clustername="<Cluster_Name>" \
    -n mongodb-atlas-test
secret/mongo-atlas-secret created
```

### Populate data in database

To insert data into database, you need to provide `connection string`.  You can
find this `connection string` using steps mentioned [here](https://www.mongodb.com/docs/atlas/tutorial/connect-to-your-cluster/#connect-to-your-atlas-cluster)

```bash
# Create a collection in database
$ mongosh "<connection string>" --apiVersion 1 \
    --username <database username> -p <database password> \
    --quiet --eval "db.people.insertOne({'name': {'first': 'Alan', last: 'Turing'}})"
{
  acknowledged: true,
  insertedId: ObjectId("643812344c2ce812b7aeaccf")
}

# View the people data in the database
$ mongosh "<connection string>" --apiVersion 1 \
    --username <database username> -p <database password> \
    --quiet --eval "db.people.find()"
[
  {
    _id: ObjectId("643812344c2ce812b7aeaccf"),
    name: { first: 'Alan', last: 'Turing' }
  }
]
```

## Protect the Application

You can now take a backup of the MongoDB data using an ActionSet defining
backup for this application. Create an ActionSet in the same namespace as the
controller.

```bash
$ kanctl create actionset --action backup --namespace kanister \
    --blueprint mongodb-atlas-blueprint \
    --objects v1/secrets/mongodb-atlas-test/mongo-atlas-secret
actionset backup-tfjps created

# View the status of the actionset
$ kubectl --namespace kanister get actionsets.cr.kanister.io backup-tfjps
NAME           PROGRESS   RUNNING PHASE             LAST TRANSITION TIME   STATE
backup-tfjps   100.00                               2023-04-13T14:40:24Z   complete
```

### Disaster strikes!

Let's say someone accidentally deleted the `people` collection:

```bash
# Drop the people collection
$ mongosh "<connection string>" --apiVersion 1 \
    --username <database username> -p <database password> \
    --quiet --eval "db.people.drop()"
true

# Try to access this data in the database
$ mongosh "<connection string>" --apiVersion 1 \
    --username <database username> -p <database password> \
    --quiet --eval "db.people.find()"
# No entries found
```

### Restore the Application

To restore the missing data, use the backup that was created.

```bash
$ kanctl create actionset --action restore --from backup-tfjps --namespace kanister
actionset restore-backup-tfjps-bhv5j created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io restore-backup-tfjps-bhv5j
NAME                         PROGRESS   RUNNING PHASE              LAST TRANSITION TIME   STATE
restore-backup-tfjps-bhv5j   100.00                                2023-04-13T15:07:10Z   complete
```

Now the lost data should be visible in the database.

```bash
# Try to access this data in the database
$ mongosh "<connection string>" --apiVersion 1 \
    --username <database username> -p <database password> \
    --quiet --eval "db.people.find()"
[
  {
    _id: ObjectId("643812344c2ce812b7aeaccf"),
    name: { first: 'Alan', last: 'Turing' }
  }
]
```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the
following command.

```bash
$ kanctl --namespace kanister create actionset --action delete --from backup-tfjps
actionset delete-backup-tfjps-gcjb2 created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io delete-backup-tfjps-gcjb2
NAME                        PROGRESS   RUNNING PHASE   LAST TRANSITION TIME   STATE
delete-backup-tfjps-gcjb2   100.00                     2023-04-13T15:11:24Z   complete
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset <actionset name> -n kanister
```

## Cleanup

Delete Blueprint and ActionSet

```bash
$ kubectl delete blueprints.cr.kanister.io mongodb-atlas-blueprint -n kanister
blueprint.cr.kanister.io "mongodb-atlas-blueprint" deleted

$ kubectl delete actionset backup-tfjps restore-backup-tfjps-bhv5j delete-backup-tfjps-gcjb2 -n kanister
actionset.cr.kanister.io "backup-tfjps" deleted
actionset.cr.kanister.io "restore-backup-tfjps-bhv5j" deleted
actionset.cr.kanister.io "delete-backup-tfjps-gcjb2" deleted
```
