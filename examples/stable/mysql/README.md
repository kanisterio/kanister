# MySQL

[MySQL](https://MySQL.org) is one of the most popular database servers in the world. Notable users include Wikipedia, Facebook and Google.

## Introduction

This chart bootstraps a single node MySQL deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.6+ with Beta APIs enabled
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.26.0 installed in your cluster
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
$ helm repo add stable https://kubernetes-charts.storage.googleapis.com/
$ helm repo update

$ helm install stable/mysql -n my-release --namespace mysql-test \
    --set mysqlRootPassword='asd#45@mysqlEXAMPLE' \
    --set persistence.size=10Gi
```

The command deploys an instance MySQL in `mysql-test` namespace on Kubernetes cluster with default configurations.

By default a random password will be generated for the root user. For setting your own password, use the `mysqlRootPassword` param as shown above.

You can retrieve your root password by running the following command. Make sure to replace [YOUR_RELEASE_NAME] and [YOUR_NAMESPACE]:

    `kubectl get secret [YOUR_RELEASE_NAME] -n [YOUR_NAMESPACE] -o jsonpath="{.data.mysql-root-password}" | base64 --decode`

> **Tip**: List all releases using `helm list`

## Integrating with Kanister

If you have deployed MySQL application with name other than `my-release` and namespace other than `mysql-test`, you need to modify the commands(backup, restore and delete) used below to use the correct name and namespace

### Create Profile
Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--bucket <s3-bucket-name> --region <region-name> \
	--namespace mysql-test
```

**NOTE:**

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.

### Create Blueprint

Create Blueprint in the same namespace as the controller

```bash
$ kubectl create -f ./mysql-blueprint.yaml -n kasten-io
```

Once MySQL is running, you can populate it with some data. Let's add a table called "pets" to a test database:

```bash
# Connect to MySQL by running a shell inside MySQL's pod
$ kubectl exec -ti $(kubectl get pods -n mysql-test --selector=app=my-release-mysql -o=jsonpath='{.items[0].metadata.name}') -n mysql-test -- bash

# From inside the shell, use the mysql CLI to insert some data into the test database
# Create "test" db

$ mysql --user=root --password=<mysql-root-password>

mysql> CREATE DATABASE test;
Query OK, 1 row affected (0.00 sec)

mysql> USE test;
Database changed

# Create "pets" table
mysql> CREATE TABLE pets (name VARCHAR(20), owner VARCHAR(20), species VARCHAR(20), sex CHAR(1), birth DATE, death DATE);
Query OK, 0 rows affected (0.02 sec)

# Insert row to the table
mysql> INSERT INTO pets VALUES ('Puffball','Diane','hamster','f','1999-03-30',NULL);
Query OK, 1 row affected (0.01 sec)

# View data in "pets" table
mysql> SELECT * FROM pets;
+----------+-------+---------+------+------------+-------+
| name     | owner | species | sex  | birth      | death |
+----------+-------+---------+------+------------+-------+
| Puffball | Diane | hamster | f    | 1999-03-30 | NULL  |
+----------+-------+---------+------+------------+-------+
1 row in set (0.00 sec)
```

## Protect the Application

You can now take a backup of the MySQL data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
# Find profile name
$ kubectl get profile -n mysql-test
NAME               AGE
s3-profile-drnw9   2m

# Create Actionset
$ kanctl create actionset --action backup --namespace kasten-io --blueprint mysql-blueprint --deployment mysql-test/my-release-mysql --profile mysql-test/s3-profile-drnw9 --secrets mysql=mysql-test/my-release-mysql
actionset backup-rslmb created

$ kubectl --namespace kasten-io get actionsets.cr.kanister.io
NAME                 AGE
backup-rslmb         1m

# View the status of the actionset
$ kubectl --namespace kasten-io describe actionset backup-rslmb
```

### Disaster strikes!

Let's say someone accidentally deleted the test database using the following command:
```bash
# Connect to MySQL by running a shell inside MySQL's pod
$ kubectl exec -ti $(kubectl get pods -n mysql-test --selector=app=my-release-mysql -o=jsonpath='{.items[0].metadata.name}') -n mysql-test -- bash

$ mysql --user=root --password=asd#45@mysqlEXAMPLE

# Drop the test database
$ mysql> SHOW DATABASES;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| mysql              |
| performance_schema |
| sys                |
| test               |
+--------------------+
5 rows in set (0.00 sec)

mysql> DROP DATABASE test;
Query OK, 1 row affected (0.03 sec)

mysql> SHOW DATABASES;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| mysql              |
| performance_schema |
| sys                |
+--------------------+
4 rows in set (0.00 sec)

```

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
$ kanctl --namespace kasten-io create actionset --action restore --from "backup-rslmb"
actionset restore-backup-62vxm-2hdsz created

# View the status of the ActionSet
$ kubectl --namespace kasten-io describe actionset restore-backup-62vxm-2hdsz
```

Once the ActionSet status is set to "complete", you can see that the data has been successfully restored to MySQL

```bash
mysql> SHOW DATABASES;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| mysql              |
| performance_schema |
| sys                |
| test               |
+--------------------+
5 rows in set (0.00 sec)

mysql> USE test;
Reading table information for completion of table and column names
You can turn off this feature to get a quicker startup with -A

Database changed
mysql> SHOW TABLES;
+----------------+
| Tables_in_test |
+----------------+
| pets           |
+----------------+
1 row in set (0.00 sec)

mysql> SELECT * FROM pets;
+----------+-------+---------+------+------------+-------+
| name     | owner | species | sex  | birth      | death |
+----------+-------+---------+------+------------+-------+
| Puffball | Diane | hamster | f    | 1999-03-30 | NULL  |
+----------+-------+---------+------+------------+-------+
1 row in set (0.00 sec)

```


## Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kasten-io logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-62vxm-2hdsz -n kasten-io
```


## Cleanup

### Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release --purge
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

### Delete CRs
Remove Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io mysql-blueprint -n kasten-io

$ kubectl get profiles.cr.kanister.io -n mysql-test
NAME               AGE
s3-profile-drnw9   122m
$ kubectl delete profiles.cr.kanister.io s3-profile-drnw9 -n mysql-test
```
