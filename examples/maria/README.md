# MariaDB

[MariaDB](https://mariadb.org/) is open source relation database that that is made by the developers of MySQL.

## Introduction

This chart bootstraps a single node MariaDB deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.6+ with Beta APIs enabled
- PV provisioner support in the underlying infrastructure
- Kanister controller installed in your cluster, let's assume in Namespace `kanister`
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)

## Installing the Chart

To install the Maria database using the `bitnami` chart with the release name `my-release`:

```bash
# Add bitnami in your local chart repository
$ helm repo add bitnami https://charts.bitnami.com/bitnami

# Update your local chart repository
$ helm repo update

# Install the maria database
$ kubectl create namespace maria-db
$ helm install my-release bitnami/mariadb -n maria-db

```

The command deploys a MariaDB instance in the `maria-db` namespace.


You can retrieve your root password by running the following command. Make sure to replace [YOUR_RELEASE_NAME] and [YOUR_NAMESPACE]:

    `kubectl get secret <release-name>-mariadb -n <release-ns> -ojsonpath="{.data.mariadb-root-password}" | base64 --decode`

> **Tip**: List all releases using `helm list --all-namespaces`.

## Integrating with Kanister

If you have deployed MariaDB application with name other than `my-release` and namespace other than `maria-db`, you need to modify the commands(backup, restore and delete) used below to use the correct release name and namespace

### Create Profile
Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--bucket <s3-bucket-name> --region <region-name> \
	--namespace maria-db
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

```bash
$ kubectl create -f maria-blueprint.yaml -n kanister
```

Once MariaDB is running, you can populate it with some data. Let's add a table called "pets" to a test database:

```bash
# Connect to MariaDB by running a shell inside MariaDB's pod
$ kubectl exec -ti my-release-mariadb-0 -n maria-db -- bash

# From inside the shell, use the mysql CLI to insert some data into the test database
# Create "test" db

# Replace maria-root-password with the password that you have set while installing MariaDB
# or you can get it from the secret that is created in the maria-db namespace, named `my-release-mariadb`
$ mysql --user=root --password=<maria-root-password>

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

You can now take a backup of the MariaDB data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
# Find profile name
$ kubectl get profile -n maria-db
NAME               AGE
s3-profile-z75pl   2m

# Create Actionset
# Please make sure the value of profile and blueprint matches with the names of profile and blueprint that we have created already
$ kanctl create actionset --action backup --namespace kanister --blueprint maria-blueprint --statefulset maria-db/my-release-mariadb --profile maria-db/s3-profile-z75pl
actionset backup-8q2kx created

$ kubectl --namespace kasten-io get actionsets.cr.kanister.io
NAME                 AGE
backup-8q2kx         1m

# View the status of the actionset
# Please make sure the name of the actionset here matches with name of the name of actionset that we have created already
$ kubectl --namespace kanister describe actionset backup-8q2kx
```

### Disaster strikes!

Let's say someone accidentally deleted the test database using the following command:
```bash
# Connect to MariaDB by running a shell inside MariaDB's pod
$ kubectl exec -ti my-release-mariadb-0 -n maria-db -- bash

$ mysql --user=root --password=<mariadb-root-password>

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
# Make sure to use correct backup actionset name here
$ kanctl --namespace kanister create actionset --action restore --from "backup-8q2kx"
actionset restore-backup-8q2kx-2hdsz created

# View the status of the ActionSet
# Make sure to use correct restore actionset name here
$ kubectl --namespace kanister describe actionset restore-backup-62vxm-2hdsz
```

Once the ActionSet status is set to "complete", you can see that the data has been successfully restored to MariaDB

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
$ kubectl --namespace kanister logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-8q2kx-2hdsz -n kanister
```


## Cleanup

### Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
# Helm Version 3
$ helm delete my-release -n maria-db
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

### Delete CRs
Remove Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io maria-blueprint -n kanister

$ kubectl get profiles.cr.kanister.io -n maria-db
NAME               AGE
s3-profile-z75pl   122m

$ kubectl delete profiles.cr.kanister.io s3-profile-z75pl -n maria-db
```
