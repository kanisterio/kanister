# Microsoft SQL Server

MS SQL Server is a relational database management system (RDBMS) developed by Microsoft. This product is built for the basic function of storing retrieving data as required by other applications.

## Introduction
This document will cover how to install SQL Server and how to run backup/restore actions.

## Prerequisites

- Kubernetes 1.16+ with Beta APIs enabled
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.110.0 installed in your cluster, let's assume in Namespace `kanister`
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)

## Installing Microsoft SQL Server

Resources created as part of installation
- PVC
- Deployment
- Service

### Create Namespace

```bash
$ kubectl create ns sqlserver
```

### Create Password

```bash
$ kubectl create secret generic mssql --from-literal=SA_PASSWORD="MyC0m9l&xP@ssw0rd" -n sqlserver
```

### Create storage
Execute following commands to create PVC for SQL Server installation.
Default storage class will be used to provision PVC.
```bash
$ cat <<EOF | kubectl create -f -
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: mssql-data
  namespace: sqlserver
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 8Gi
EOF
```

### Create Deployment
Create service and deployment by using following code

```bash
$ cat <<EOF | kubectl create -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mssql-deployment
  namespace: sqlserver
  labels:
    app: mssql
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mssql
  template:
    metadata:
      labels:
        app: mssql
    spec:
      terminationGracePeriodSeconds: 30
      hostname: mssqlinst
      securityContext:
        fsGroup: 10001
      containers:
        - name: mssql
          image: mcr.microsoft.com/mssql/server:2019-CU27-ubuntu-20.04
          ports:
            - containerPort: 1433
          env:
            - name: MSSQL_PID
              value: "Developer"
            - name: ACCEPT_EULA
              value: "Y"
            - name: SA_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: mssql
                  key: SA_PASSWORD
          volumeMounts:
            - name: mssqldb
              mountPath: /var/opt/mssql
      volumes:
        - name: mssqldb
          persistentVolumeClaim:
            claimName: mssql-data
---
apiVersion: v1
kind: Service
metadata:
  name: mssql-deployment
  namespace: sqlserver
spec:
  selector:
    app: mssql
  ports:
    - protocol: TCP
      port: 1433
      targetPort: 1433
  type: ClusterIP
EOF
```

## Create Database

After creating server let's create database with some data in it.
SQL server comes equiped with command line utility called `sqlcmd`.
Let's use this utility to interact with database.

```bash
$ kubectl exec -it -n sqlserver $(kubectl get pods --selector=app=mssql -o=jsonpath='{.items[0].metadata.name}' -n sqlserver) -- bash

# Once inside the container, login using sqlcmd cli.
$ /opt/mssql-tools/bin/sqlcmd -S localhost -U SA -P "MyC0m9l&xP@ssw0rd"

# Execute the following script to create a database named TestDB and add table Inventory with some data.
# Create database "TestDB"
1> CREATE DATABASE TestDB
2> SELECT Name from sys.Databases
3> GO
Name
---------------------------------------------------------------------------
master
tempdb
model
msdb
TestDB

# Create table "Inventory" inside database "TestDB"
1> USE TestDB
2> CREATE TABLE Inventory (id INT, name NVARCHAR(50), quantity INT)
3> INSERT INTO Inventory VALUES (1, 'banana', 150); INSERT INTO Inventory VALUES (2, 'orange', 154);
4> GO

(1 rows affected)

(1 rows affected)

# View data in "Inventory" table
1> SELECT * FROM Inventory;
2> GO
id          name                                               quantity
----------- -------------------------------------------------- -----------
          1 banana                                                     150
          2 orange                                                     154

```
After following all given steps database named `TestDB` should have table called `Inventory`

## Integrating with Kanister

If you have deployed SQL Server with name other than `mssql-deployment` and namespace other than `sqlserver`,
you need to modify the commands(backup, restore and delete) used below to use the correct release name and namespace

### Create Profile
Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--bucket <s3-bucket-name> \
	--region <region-name> \
	--namespace kanister
```
You can read more about the Profile custom Kanister resource [here](https://docs.kanister.io/architecture.html?highlight=profile#profiles).

**NOTE:**

The above command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.

### Create Blueprint

**NOTE: v2 Blueprints are experimental and are not supported with standalone Kanister.**

Create Blueprint in the same namespace as the Kanister controller

Execute following command to create the blueprint
```bash
$ kubectl create -f ./mssql-blueprint.yaml -n kanister
```
Blueprint with name `mssql-blueprint` will be created in `kanister` namespace

## Protect the Application

You can now take a backup of the SQL Server data using an ActionSet defining backup for this application.
Create an ActionSet in the same namespace as the controller.

```bash
# Find profile name
$ kubectl get profile -n kanister
NAME               AGE
s3-profile-k4r8w   7d1h

# Create Actionset
# Please make sure the value of profile and blueprint matches with the names of profile and blueprint that we have created already
$ kanctl create actionset --action backup --namespace kanister --blueprint mssql-blueprint --profile s3-profile-k4r8w --secrets mssql=sqlserver/mssql --deployment sqlserver/mssql-deployment
actionset backup-dzchc created


$ kubectl get actionsets -n kanister
NAME                         AGE
backup-dzchc                 29s

# View the status of the actionset
# Please make sure the name of the actionset here matches with the name of actionset that we have created above and make sure the status is complete.
$ kubectl describe actionset backup-dzchc -n kanister
```

### Disaster strikes!

Let's say someone accidentally deleted the test database using the following command:

```bash
# Connect to SQL Sever by running a shell inside mssql pod
$ kubectl exec -it -n sqlserver $(kubectl get pods --selector=app=mssql -o=jsonpath='{.items[0].metadata.name}' -n sqlserver) -- bash

$ /opt/mssql-tools/bin/sqlcmd -S localhost -U SA -P "MyC0m9l&xP@ssw0rd"

1> SELECT Name from sys.Databases
2> GO
Name
---------------------------------------------------------------------------
master
tempdb
model
msdb
TestDB

# Drop database "TestDB"
1> DROP DATABASE TestDB
2> GO

# View list of databases available
1> SELECT Name from sys.Databases
2> go
Name
--------------------------------------------------------------------------------------------------------------------------------
master
tempdb
model
msdb

```

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
# Make sure to use correct backup actionset name here
$ kanctl create actionset --action restore --namespace kanister --from "backup-dzchc"
actionset restore-backup-dzchc-vqr5v created

# View the status of the ActionSet
# Make sure to use correct restore actionset name here
$ kubectl describe actionset restore-backup-dzchc-vqr5v -n kanister
```

Once the ActionSet status is set to "complete", you can see that the data has been successfully restored to SQL Server.

```bash
1> SELECT Name from sys.Databases
2> GO
Name
---------------------------------------------------------------------------
master
tempdb
model
msdb
TestDB

# View data in "Inventory" table
1> USE TestDB
2> SELECT * FROM Inventory;
3> GO
id          name                                               quantity
----------- -------------------------------------------------- -----------
          1 banana                                                     150
          2 orange                                                     154

```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl create actionset --action delete --namespace kanister --from "backup-dzchc"
actionset delete-backup-dzchc-kcvkg created

# View the status of the ActionSet
$ kubectl describe actionset delete-backup-dzchc-kcvkg -n kanister
```

## Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

You can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-dzchc-vqr5v -n kanister
```

## Cleanup

### Uninstalling the SQL Server

To uninstall/delete the `mssql` deployment:

```bash
# Delete deployment, service and pvc
$ kubectl delete service/mssql-deployment -n sqlserver
$ kubectl delete deployment.apps/mssql-deployment -n sqlserver
$ kubectl delete pvc mssql-data -n sqlserver
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

### Delete CRs
Remove Blueprint and Profile CR

```bash
$ kubectl delete blueprint mssql-blueprint -n kanister
blueprint.cr.kanister.io "mssql-blueprint" deleted

$ kubectl get profiles -n kanister
NAME               AGE
s3-profile-k4r8w   122m

$ kubectl delete profile s3-profile-k4r8w -n kanister
profile.cr.kanister.io "s3-profile-k4r8w" deleted
```
