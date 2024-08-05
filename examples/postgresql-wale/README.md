# PostgreSQL

[PostgreSQL](https://www.postgresql.org/) is an object-relational database management system (ORDBMS) with an emphasis on extensibility and on standards-compliance.

## Introduction

This chart bootstraps a [PostgreSQL](https://github.com/bitnami/bitnami-docker-postgresql) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

Bitnami charts can be used with [Kubeapps](https://kubeapps.com/) for deployment and management of Helm Charts in clusters. This chart has been tested to work with NGINX Ingress, cert-manager, fluentd and Prometheus on top of the [BKPR](https://kubeprod.io/).

## Prerequisites

- Kubernetes 1.10+
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.110.0 installed in your cluster
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## Installing the Chart
To install the chart with the release name `my-release`:

```bash
$ helm repo add bitnami https://charts.bitnami.com/bitnami
$ helm repo update

$ helm install my-release bitnami/postgresql  \
	--namespace postgres-test --create-namespace \
	--set image.repository=ghcr.io/kanisterio/postgresql \
	--set image.tag=0.110.0 \
	--set postgresqlPassword=postgres-12345 \
	--set postgresqlExtendedConf.archiveCommand="'envdir /bitnami/postgresql/data/env wal-e wal-push %p'" \
	--set postgresqlExtendedConf.archiveMode=true \
	--set postgresqlExtendedConf.archiveTimeout=60 \
	--set postgresqlExtendedConf.walLevel=archive
```

The command deploys PostgreSQL on the Kubernetes cluster in the default configuration.

> **Tip**: List all releases using `helm list`

In case, if you don't have `Kanister` installed already, you can use following commands to do that.
Add Kanister Helm repository and install Kanister operator
```bash
$ helm repo add kanister https://charts.kanister.io
$ helm install kanister --namespace kanister --create-namespace kanister/kanister-operator --set image.tag=0.110.0
```

## Integrating with Kanister

If you have deployed postgresql application with name other than `my-release` and namespace other than `postgres-test`, you need to modify the commands used below to use the correct name and namespace

### Create Profile

Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--bucket <s3-bucket-name> --region <region-name> \
	--namespace postgres-test
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
$ kubectl create -f ./postgresql-blueprint.yaml -n kanister
```

### Create a Base Backup
Create an ActionSet in the same namespace as the controller to trigger a backup. This will also setup log shipping that enables restoring to point-in-time restore

```bash
# Find profile name
$ kubectl get profile -n postgres-test
NAME               AGE
s3-profile-zvrg9   109m

# Create Actionset
# Create a base backup by creating an ActionSet
cat << EOF | kubectl create -f -
apiVersion: cr.kanister.io/v1alpha1
kind: ActionSet
metadata:
    name: pg-base-backup
    namespace: kanister
spec:
    actions:
    - name: backup
      blueprint: postgresql-blueprint
      object:
        kind: StatefulSet
        name: my-release-postgresql
        namespace: postgres-test
      profile:
        apiVersion: v1alpha1
        kind: Profile
        name: s3-profile-k8s9l
        namespace: postgres-test
      secrets:
        postgresql:
          name: my-release-postgresql
          namespace: postgres-test
EOF

# View the status of the actionset
$ kubectl --namespace kanister describe actionset pg-base-backup
```



Once Postgres is running, you can populate it with some data. Let's add a table called "company" to a "test" database:
```
## Log in into postgresql container and get shell access
$ kubectl exec -ti my-release-postgresql-0 -n postgres-test -- bash

## use psql cli to add entries in postgresql database
$ PGPASSWORD=${POSTGRES_PASSWORD} psql
psql (11.5)
Type "help" for help.

## Create DATABASE
postgres=# CREATE DATABASE test;
CREATE DATABASE
postgres=# \l
                                  List of databases
   Name    |  Owner   | Encoding |   Collate   |    Ctype    |   Access privileges
-----------+----------+----------+-------------+-------------+-----------------------
 postgres  | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
 template0 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
           |          |          |             |             | postgres=CTc/postgres
 template1 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
           |          |          |             |             | postgres=CTc/postgres
 test      | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
(4 rows)

## Create table COMPANY in test database
postgres=# \c test
You are now connected to database "test" as user "postgres".
test=# CREATE TABLE COMPANY(
test(#     ID INT PRIMARY KEY     NOT NULL,
test(#     NAME           TEXT    NOT NULL,
test(#     AGE            INT     NOT NULL,
test(#     ADDRESS        CHAR(50),
test(#     SALARY         REAL,
test(#     CREATED_AT    TIMESTAMP
test(# );
CREATE TABLE

## Insert data into the table
test=# INSERT INTO COMPANY (ID,NAME,AGE,ADDRESS,SALARY,CREATED_AT) VALUES (10, 'Paul', 32, 'California', 20000.00, now());
INSERT 0 1
test=# select * from company;
 id | name | age |                      address                       | salary |         created_at
----+------+-----+----------------------------------------------------+--------+----------------------------
 10 | Paul |  32 | California                                         |  20000 | 2019-09-16 14:39:36.316065
(1 row)

## Add few more entries
test=# INSERT INTO COMPANY (ID,NAME,AGE,ADDRESS,SALARY,CREATED_AT) VALUES (20, 'Omkar', 32, 'California', 20000.00, now());
INSERT 0 1
test=# INSERT INTO COMPANY (ID,NAME,AGE,ADDRESS,SALARY,CREATED_AT) VALUES (30, 'Prasad', 32, 'California', 20000.00, now());
INSERT 0 1

test=# select * from company;
 id | name  | age |                      address                       | salary |         created_at
----+-------+-----+----------------------------------------------------+--------+----------------------------
 10 | Paul  |  32 | California                                         |  20000 | 2019-09-16 14:39:36.316065
 20 | Omkar |  32 | California                                         |  20000 | 2019-09-16 14:40:52.952459
 30 | Omkar |  32 | California                                         |  20000 | 2019-09-16 14:41:06.433487
```

### Disaster strikes!

Let's say someone accidentally deleted the test database using the following command:

```bash
## Log in into postgresql container and get shell access
$ kubectl exec -ti my-release-postgresql-0 -n postgres-test -- bash

## use psql cli to add entries in postgresql database
$ PGPASSWORD=${POSTGRES_PASSWORD} psql
psql (11.5)
Type "help" for help.

## Drop database
postgres=# \l
                                  List of databases
   Name    |  Owner   | Encoding |   Collate   |    Ctype    |   Access privileges
-----------+----------+----------+-------------+-------------+-----------------------
 postgres  | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
 template0 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
           |          |          |             |             | postgres=CTc/postgres
 template1 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
           |          |          |             |             | postgres=CTc/postgres
 test      | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
(4 rows)

postgres=# DROP DATABASE test;
DROP DATABASE
postgres=# \l
                                  List of databases
   Name    |  Owner   | Encoding |   Collate   |    Ctype    |   Access privileges
-----------+----------+----------+-------------+-------------+-----------------------
 postgres  | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
 template0 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
           |          |          |             |             | postgres=CTc/postgres
 template1 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
           |          |          |             |             | postgres=CTc/postgres
(3 rows)
```

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

Let's use PostgreSQL Point-In-Time Recovery to recover data till perticular time

```bash
$ kanctl --namespace kanister create actionset --action restore --from pg-base-backup --options pitr=2019-09-16T14:41:00Z
actionset restore-pg-base-backup-d7g7w created

## NOTE: pitr argument to the command is optional. If you want to restore data till the latest consistent state, you can skip '--options pitr' option
# e.g $ kanctl --namespace kanister create actionset --action restore --from pg-base-backup

## Check status
$ kubectl --namespace kanister describe actionset restore-pg-base-backup-d7g7w
```

Once the ActionSet status is set to "complete", you can see that the data has been successfully restored to PostgreSQL

```bash
postgres=# \l
                                  List of databases
   Name    |  Owner   | Encoding |   Collate   |    Ctype    |   Access privileges
-----------+----------+----------+-------------+-------------+-----------------------
 postgres  | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
 template0 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
           |          |          |             |             | postgres=CTc/postgres
 template1 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 | =c/postgres          +
           |          |          |             |             | postgres=CTc/postgres
 test      | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |
(4 rows)

postgres=# \c test;
You are now connected to database "test" as user "postgres".
test=# select * from company;
 id | name  | age |                      address                       | salary |         created_at
----+-------+-----+----------------------------------------------------+--------+----------------------------
 10 | Paul  |  32 | California                                         |  20000 | 2019-09-16 14:39:36.316065
 20 | Omkar |  32 | California                                         |  20000 | 2019-09-16 14:40:52.952459

(2 rows)


```

## Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-md6gb-d7g7w -n kanister
```

## Cleanup

### Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```console
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.
To completely remove the release include the `--purge` flag.

### Delete CRs
Remove Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io postgresql-blueprint -n kanister

$ kubectl get profiles.cr.kanister.io -n postgres-test
NAME               AGE
s3-profile-zvrg9   125m
$ kubectl delete profiles.cr.kanister.io s3-profile-zvrg9 -n postgres-test
```
