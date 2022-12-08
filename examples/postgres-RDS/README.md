# Postgres-RDS

[PostgreSQL](https://www.postgresql.org/) is an object-relational database management system (ORDBMS) with an emphasis on extensibility and standards compliance.

## Introduction

This example is to demonstrate how Kanister can be used to provision AWS RDS Instance and import data into it from On-prem Postgres using Kanister functions

## Prerequisites

- Kubernetes 1.10+
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.84.0 installed in your cluster
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## Create RDS instance on AWS

We need to have a Postgres RDS instance where we would restore the data.

> You can skip this step if you already have an RDS instance created

RDS instance needs to be reachable from the outside world. So make sure that you have VPC with the security group having the rule to allow ingress traffic on 5432 TCP port.


You can create a security group and add rules to the default VPC using the following commands

**NOTE**

This is highly insecure. Its good only for testing and shouldn't be use in a production environment.

```bash
aws ec2 create-security-group --group-name pgtest-sg --description "pgtest security group"
aws ec2 authorize-security-group-ingress --group-name pgtest-sg --protocol tcp --port 5432 --cidr 0.0.0.0/0
```

Now create an RDS instance with the PostgreSQL engine

```bash
aws rds create-db-instance \
    --publicly-accessible \
    --allocated-storage 20 --db-instance-class db.t3.micro \
    --db-instance-identifier test-postgresql-instance \
    --engine postgres \
    --master-username master \
    --vpc-security-group-id sg-xxxxyyyyzzz \ # Sec group with TCP 5432 inbound rule
    --master-user-password secret99

aws rds wait db-instance-available --db-instance-identifier=test-postgresql-instance
```
## Create configmap

Create a configmap that contains information to connect to the RDS DB instance

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: dbconfig
data:
  region: us-west-2 #Region in which instance is to be created
  instance_name: test-postgresql-instance  # name of the RDS Instance that would be provisioned
```

## Create secret

Create a secret that contains credentials to connect to the RDS DB instance

```
apiVersion: v1
kind: Secret
metadata:
  name: dbsecret
stringData:
  ########## AWS Key CREDS
  accessKeyId: 
  secretAccessKey:
  ########## Database instance creds
  postgres_username: master
  postgres_password: secret99 
```

**NOTE**

The creds must have proper permissions to perform operation on RDS

## Installing the PostgreSQL Chart
To install the chart with the release name `my-release`:

```bash
$ helm repo add bitnami https://charts.bitnami.com/bitnami
$ helm repo update

$ kubectl create ns postgres-test
$ helm install my-release --namespace postgres-test bitnami/postgresql
```

The command deploys PostgreSQL on the Kubernetes cluster in the default configuration.

> **Tip**: List all releases using `helm list`

In case, if you don't have `Kanister` installed already, you can use the following commands to do that.
Add Kanister Helm repository and install Kanister operator
```bash
$ helm repo add kanister https://charts.kanister.io
$ helm install kanister --namespace kanister --create-namespace kanister/kanister-operator --set image.tag=0.84.0
```

## Integrating with Kanister

If you have deployed a PostgreSQL application with a name other than `my-release` and a namespace other than `postgres-test`, you need to modify the commands used below to use the correct name and namespace

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
Create a Blueprint in the same namespace as the controller

```bash
$ kubectl create -f ./rds-restore-blueprint.yaml -n kanister
```

Once Postgres is running, you can populate it with some data. Let's add a table called "company" to a "test" database:
```
## Login to PostgreSQL container and get shell access
$ kubectl exec -ti my-release-postgresql-0 -n postgres-test -- bash

## use psql CLI to add entries in the PostgreSQL database
$ PGPASSWORD=${POSTGRES_PASSWORD} psql -U postgres
psql (14.0)
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
     ID INT PRIMARY KEY     NOT NULL,
     NAME           TEXT    NOT NULL,
     AGE            INT     NOT NULL,
     ADDRESS        CHAR(50),
     SALARY         REAL,
     CREATED_AT    TIMESTAMP
);
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

## Protect the Application

You can now take a backup of the PostgresDB data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
$ kubectl get profile -n postgres-test
NAME               AGE
s3-profile-7d6wt   7m25s

$ kanctl create actionset --action backup --namespace kanister --blueprint postgres-bp --statefulset postgres-test/my-release-postgresql --profile postgres-test/s3-profile-7d6wt
actionset backup-llfb8 created

$ kubectl --namespace kanister get actionsets.cr.kanister.io
NAME           AGE
backup-glptq   38s

# View the status of the actionset
$ kubectl --namespace kanister describe actionset backup-glptq
```

### Restore the Application to RDS Database instance

To restore the data into RDS Postgres instance, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets. Apart from this we also need to pass RDS instance details to the blueprint, the secret and configmap resource containing these details can be passed to kanctl:

```bash
$ kanctl --namespace kanister create actionset --action restore --from backup-glptq --config-maps dbconfig=postgres-test/dbconfig --secrets dbsecret=postgres-test/dbsecret
actionset restore-backup-glptq-6jzt4 created

## Check status
$ kubectl --namespace kanister describe actionset restore-backup-glptq-6jzt4
```

Once the ActionSet status is set to "complete", you can see that the data has been successfully restored to the RDS PostgreSQL database instance

To verify, Connect to the RDS PostgreSQL database instance using `psql`

PGPASSWORD="secret99" psql --host test-postgresql-instance.cjbctojw4ahh.us-west-2.rds.amazonaws.com -U master -d postgres -p 5432

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
 id |  name  | age |                      address                       | salary |         created_at
----+--------+-----+----------------------------------------------------+--------+----------------------------
 10 | Paul   |  32 | California                                         |  20000 | 2019-12-23 07:13:10.459499
 20 | Omkar  |  32 | California                                         |  20000 | 2019-12-23 07:13:20.953172
 30 | Prasad |  32 | California                                         |  20000 | 2019-12-23 07:13:29.15668
(3 rows)
```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from backup-glptq
actionset delete-backup-glptq-cq6bw created

# View the status of the ActionSet
$ kubectl --namespace kanister describe actionset delete-backup-glptq-cq6bw
```

## Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

you can also check the events of the actionset

```bash
$ kubectl describe actionset <actionset-name> -n kanister
```

## Cleanup

### Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash

$ helm delete my-release -n postgres-test
```

### Delete CRs
Remove Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io postgres-bp -n kanister

$ kubectl get profiles.cr.kanister.io -n postgres-test
NAME               AGE
s3-profile-7d6wt   17m
$ kubectl delete profiles.cr.kanister.io ss3-profile-7d6w -n postgres-test
```