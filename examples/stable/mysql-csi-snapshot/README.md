# Backup and restore MySQL DB using CSI VolumeSnapshot

VolumeSnapshots provide Kubernetes users with a standardized way to copy a volume's contents at a particular point in time without creating an entirely new volume. This functionality enables, for example, database administrators to backup databases before performing edit or delete modifications.

## Introduction
This document explains how Kanister leverages the use of CSI VolumeSnapshots to take backup and restore of a database in MySQL.

## Prerequisites

- Helm 3 installed
- Kubernetes 1.16+ with Beta APIs enabled
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.71.0 installed in your cluster, let's assume in Namespace `kanister`
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)
- CSI Driver installed
- VolumeSnapshotClass and StorageClass resources defined (uses above CSI Driver)
- Snapshot APIs enabled

## Install MySQL

Install the MySQL database using the bitnami chart with the release name `mysql-release`.

```bash
# Add bitnami in your local chart repository
$ helm repo add bitnami https://charts.bitnami.com/bitnami

# Update your local chart repository
$ helm repo update

# Install the MySQL database (Helm Version 3)
$ kubectl create namespace mysql
$ helm install mysql-release bitnami/mysql --namespace mysql \
    --set auth.rootPassword='asd#45@mysqlEXAMPLE' \
    --set architecture="standalone"
```

Above command deploys a MySQL instance in the `mysql` namespace.

To retrieve your root password run the following command.

```bash
$ kubectl get secret mysql-release --namespace mysql -o jsonpath="{.data.mysql-root-password}" | base64 --decode`
```

> **Tip**: List all releases using `helm list --all-namespaces`, using Helm Version 3.

# Create Application data

Connect to the MySQL database.

```bash
# Run shell inside MySQL's pod
$ kubectl exec -it $(kubectl get pods -n mysql --selector=app.kubernetes.io/instance=mysql-release -o=jsonpath='{.items[0].metadata.name}') -n mysql -- bash
```

From inside the shell, use the mysql CLI to insert some data into the test database.

```bash
$ mysql --user=root --password=asd#45@mysqlEXAMPLE

mysql> CREATE DATABASE test;
Query OK, 1 row affected (0.00 sec)

mysql> USE test;
Database changed

# Create "employees" table
mysql> CREATE TABLE employees (name VARCHAR(20), manager VARCHAR(20), address VARCHAR(200), sex CHAR(1), birth DATE, department VARCHAR(20));
Query OK, 0 rows affected (0.02 sec)

# Insert row to the table
mysql> INSERT INTO employees VALUES ('Max','Tom','Naperville, Illinois, Chicago','m','1999-03-30','Engineering');
Query OK, 1 row affected (0.01 sec)

# View data in "employees" table
mysql> SELECT * FROM employees;
+------+---------+-------------------------------+------+------------+-------------+
| name | manager | address                       | sex  | birth      | department  |
+------+---------+-------------------------------+------+------------+-------------+
| Max  | Tom     | Naperville, Illinois, Chicago | m    | 1999-03-30 | Engineering |
+------+---------+-------------------------------+------+------------+-------------+
1 row in set (0.00 sec)
```

# Backup MySQL DB

## Create Blueprint

Create Blueprint in the same namespace as the Kanister controller.

> **Note**: We used a Kubernetes cluster on DigitalOcean. Hence, snapshotClass and storageClass in the ./mysql-csi-snapshot-bp.yaml file is set to `do-block-storage`. Please correct these arguments as per your cluster setup. Either before creating the blueprint or after creating with the help of `kubectl patch` or `kubectl edit` commands.

```bash
$ kubectl create -f ./mysql-csi-snapshot-bp.yaml -n kanister
```

## Backup the application data

Take a backup of the MySQL data using the backup ActionSet from above blueprint. Create an ActionSet in the `kanister` namespace. An easy way to do this is by using `kanctl`.

```bash
# Create Actionset
# Please make sure the value of blueprint matches with the name of blueprint that we created previously
$ kanctl create actionset --action backup --namespace kanister --blueprint mysql-csi-snapshot-bp --pvc mysql/$(kubectl get pvc -n mysql --selector=app.kubernetes.io/instance=mysql-release -o=jsonpath='{.items[0].metadata.name}')
actionset backup-mlvcv created

$ kubectl --namespace kanister get actionset
NAME                         AGE
backup-mlvcv                 112s

# View the status of the actionset
# Please make sure the name of the actionset here matches with name of the name of actionset that we have created already
$ kubectl --namespace kanister describe actionset backup-mlvcv

# Check the CSI VolumeSnapshot created
$ kubectl -n mysql get volumesnapshot
NAME                                  READYTOUSE   SOURCEPVC              SOURCESNAPSHOTCONTENT   RESTORESIZE   SNAPSHOTCLASS      SNAPSHOTCONTENT                                    CREATIONTIME   AGE
data-mysql-release-0-snapshot-cd26z   true         data-mysql-release-0                           8Gi           do-block-storage   snapcontent-2e411d1d-0b5f-48d4-9b79-b19e824c2e38   3h40m          3h40m
```

# Disaster strikes!

Let's say someone accidentally deleted the test database.

```bash
# Connect to MySQL by running a shell inside MySQL's pod
$ kubectl exec -it $(kubectl get pods -n mysql --selector=app.kubernetes.io/instance=mysql-release -o=jsonpath='{.items[0].metadata.name}') -n mysql -- bash

$ mysql --user=root --password=asd#45@mysqlEXAMPLE

mysql> SHOW DATABASES;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| my_database        |
| mysql              |
| performance_schema |
| sys                |
| test               |
+--------------------+
6 rows in set (0.00 sec)

mysql> DROP DATABASE test;
Query OK, 1 row affected (0.12 sec)

mysql> SHOW DATABASES;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| my_database        |
| mysql              |
| performance_schema |
| sys                |
+--------------------+
5 rows in set (0.01 sec)
```

# Restore MySQL DB

To restore the missing data, you should use the backup that you created before. We use `kanctl` for creating this restore action. `kanctl` helps create ActionSets that depend on other ActionSets.

```bash
# Make sure to use correct backup actionset name here
$ kanctl --namespace kanister create actionset --action restore --from backup-mlvcv
actionset restore-backup-mlvcv-6z9xn created

# Check the PVCs in mysql namespace. You should see a new PVC in the format '<PVC-name>-restored'
$ kubectl -n mysql  get pvc
NAME                            STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS       AGE
data-mysql-release-0            Bound    pvc-b966c5ef-3d20-4ec2-9ab3-1a868737ea0a   8Gi        RWO            do-block-storage   4m14s
data-mysql-release-0-restored   Bound    pvc-2eb6fb89-78bc-4c2e-8c1e-c270d664f311   8Gi        RWO            do-block-storage   18s

# Use following command to debug if a new PVC in the format '<PVC-name>-restored' is not created
$ kubectl --namespace kanister describe actionset restore-backup-mlvcv-6z9xn
```

Next, we need to add the newly restored PVC `data-mysql-release-0-restored` in the `mysql-release` StatefulSet spec template. One of the easiest way to do that is using `kubectl patch` command.

```bash
$ kubectl -n mysql patch statefulset mysql-release --type=json -p='[{"op": "add", "path": "/spec/template/spec/volumes/-", "value": {"name": "restored-data", "persistentVolumeClaim": {"claimName": "data-mysql-release-0-restored"}}}]'

$ kubectl -n mysql patch statefulset mysql-release --type=json -p='[{"op": "add", "path": "/spec/template/spec/containers/0/volumeMounts", "value": [{"mountPath": "/bitnami/mysql", "name": "restored-data"}, {"mountPath": "/opt/bitnami/mysql/conf/my.cnf", "name": "config", "subPath": "my.cnf"}]}]'
```

Once the patch is complete and MySQL pod is set to “running“, you can see that the data has been successfully restored to MySQL.

# Verify the restored application data

To verify if restore was successful, we need to connect to the MySQL CLI and query the test database that we created in [this](https://github.com/kanisterio/kanister/tree/master/examples/stable/mysql-csi-snapshot#create-application-data) step.

```bash
# Enter into a shell inside MySQL's pod
$ kubectl exec -it $(kubectl get pods -n mysql --selector=app.kubernetes.io/instance=mysql-release -o=jsonpath='{.items[0].metadata.name}') -n mysql -- bash

# Login into the MySQL CLI
$ mysql --user=root --password=asd#45@mysqlEXAMPLE

mysql> SHOW DATABASES;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| my_database        |
| mysql              |
| performance_schema |
| sys                |
| test               |
+--------------------+
6 rows in set (0.01 sec)

mysql> USE test;
Database changed

# View data in "employees" table
mysql> SELECT * FROM employees;
+------+---------+-------------------------------+------+------------+-------------+
| name | manager | address                       | sex  | birth      | department  |
+------+---------+-------------------------------+------+------------+-------------+
| Max  | Tom     | Naperville, Illinois, Chicago | m    | 1999-03-30 | Engineering |
+------+---------+-------------------------------+------+------------+-------------+
1 row in set (0.00 sec)
```

# Delete the Artifacts

The CSI VolumeSnapshot created by the backup action can be cleaned up using the following command.

```bash
# Make sure to use correct backup actionset name here
$ kanctl --namespace kanister create actionset --action delete --from backup-mlvcv
actionset delete-backup-mlvcv-cq6bw created

# View the status of the ActionSet
$ kubectl --namespace kanister describe actionset delete-backup-glptq-cq6bw
```

# Cleanup

## Uninstalling the helm chart

To uninstall/delete the `mysql-release` deployment.

```bash
# Helm Version 3
$ helm delete mysql-release -n mysql

# Remove mysql namespace
$ kubectl delete namespace mysql
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Delete CRs

Remove the blueprint.

```bash
$ kubectl delete blueprints.cr.kanister.io mysql-csi-snapshot-bp -n kanister
```

Remove the actionsets.

```bash
$ kubectl delete actionsets -n kanister --all
```
