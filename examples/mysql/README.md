# MySQL

[MySQL](https://MySQL.org) is one of the most popular database servers in the world. Notable users include Wikipedia, Facebook and Google.

## Introduction

This chart bootstraps a single node MySQL deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.20+
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.110.0 installed in your cluster, let's assume in Namespace `kanister`
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)

## Installing the Chart

To install the MySQL database using the `bitnami` chart with the release name `mysql-release`:

```bash
# Add bitnami in your local chart repository
$ helm repo add bitnami https://charts.bitnami.com/bitnami

# Update your local chart repository
$ helm repo update

# Install the MySQL database
$ kubectl create namespace mysql-test
$ helm install mysql-release bitnami/mysql --namespace mysql-test \
    --set auth.rootPassword='<mysql-root-password>'
```

The command deploys a MySQL instance in the `mysql-test` namespace.

By default a random password will be generated for the root user. For setting your own password, use the `auth.rootPassword` param as shown above.

You can retrieve your root password by running the following command. Make sure to replace [YOUR_RELEASE_NAME] and [YOUR_NAMESPACE]:

    `kubectl get secret [YOUR_RELEASE_NAME] --namespace [YOUR_NAMESPACE] -o jsonpath="{.data.mysql-root-password}" | base64 --decode`

> **Tip**: List all releases using `helm list --all-namespaces`, using Helm Version 3.

## Integrating with Kanister

If you have deployed MySQL application with name other than `mysql-release` and namespace other than `mysql-test`, you need to modify the commands(backup, restore and delete) used below to use the correct release name and namespace

### Create Profile
Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--bucket <s3-bucket-name> --region <region-name> \
	--namespace mysql-test
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
$ kubectl create -f ./mysql-blueprint.yaml -n kanister
```

Once MySQL is running, you can populate it with some data. Let's add a table called "pets" to a test database:

```bash
# Connect to MySQL by running a shell inside MySQL's pod
$ kubectl exec -ti $(kubectl get pods -n mysql-test --selector=app.kubernetes.io/instance=mysql-release -o=jsonpath='{.items[0].metadata.name}') -n mysql-test -- bash

# From inside the shell, use the mysql CLI to insert some data into the test database
# Create "test" db

# Replace mysql-root-password with the password that you have set while installing MySQL
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
# Please make sure the value of profile and blueprint matches with the names of profile and blueprint that we have created already
$ kanctl create actionset --action backup --namespace kanister --blueprint mysql-blueprint --statefulset mysql-test/mysql-release --profile mysql-test/s3-profile-drnw9 --secrets mysql=mysql-test/mysql-release
actionset backup-rslmb created

# View the status of the actionset
$ kubectl --namespace kanister get actionsets.cr.kanister.io backup-rslmb
NAME           PROGRESS   LAST TRANSITION TIME   STATE
backup-rslmb   100.00     2022-12-15T09:56:49Z   complete
```

### Disaster strikes!

Let's say someone accidentally deleted the test database using the following command:
```bash
# Connect to MySQL by running a shell inside MySQL's pod
$ kubectl exec -ti $(kubectl get pods -n mysql-test --selector=app.kubernetes.io/instance=mysql-release -o=jsonpath='{.items[0].metadata.name}') -n mysql-test -- bash

$ mysql --user=root --password=<mysql-root-password>

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
$ kanctl --namespace kanister create actionset --action restore --from "backup-rslmb"
actionset restore-backup-rslmb-2hdsz created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io restore-backup-rslmb-2hdsz
NAME                         PROGRESS   LAST TRANSITION TIME   STATE
restore-backup-rslmb-2hdsz   100.00     2022-12-15T10:00:05Z   complete
```

Once the ActionSet status is set to "complete", you can see that the data has been successfully restored to MySQL

```bash
# Connect to MySQL by running a shell inside MySQL's pod
$ kubectl exec -ti $(kubectl get pods -n mysql-test --selector=app.kubernetes.io/instance=mysql-release -o=jsonpath='{.items[0].metadata.name}') -n mysql-test -- bash

mysql --user=root --password=<mysql-root-password>

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

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from backup-rslmb --namespacetargets kanister
actionset delete-backup-rslmb-cq6bw created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io delete-backup-rslmb-cq6bw
NAME                        PROGRESS   LAST TRANSITION TIME   STATE
delete-backup-rslmb-cq6bw   100.00     2022-12-15T10:05:38Z   complete
```


## Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-rslmb-2hdsz -n kanister
```


## Cleanup

### Uninstalling the Chart

To uninstall/delete the `mysql-release` deployment:

```bash
# Helm Version 3
$ helm delete mysql-release -n mysql-test
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

### Delete CRs
Remove Blueprint, Profile CR and ActionSets

```bash
$ kubectl delete blueprints.cr.kanister.io mysql-blueprint -n kanister

$ kubectl get profiles.cr.kanister.io -n mysql-test
NAME               AGE
s3-profile-drnw9   122m

$ kubectl delete profiles.cr.kanister.io s3-profile-drnw9 -n mysql-test

$ kubectl --namespace kanister delete actionsets.cr.kanister.io backup-rslmb restore-backup-rslmb-2hdsz delete-backup-rslmb-cq6bw
```
