# Postgres-RDS

[PostgreSQL](https://www.postgresql.org/) is an object-relational database management system (ORDBMS) with an emphasis on extensibility and standards compliance.

## Introduction

This example demonstrates how to provision an AWS RDS Instance, and use Kanister to backup data from an on-prem Postgres installation and restore that data into the provisioned AWS RDS instance.

## Prerequisites

- Kubernetes 1.10+
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.84.0 installed in your cluster
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## Create RDS instance on AWS

We need to have a Postgres RDS instance where we will restore the data.

> You can skip this step if you already have an RDS instance created

RDS instance needs to be reachable from the Kubernetes cluster performing the restore operation. So make sure that you have VPC with the security group having the rule to allow ingress traffic on 5432 TCP port.


You can create a security group and add rules to the default VPC using the following commands

**NOTE**

This is highly insecure. Its good only for testing and should not be used in a production environment.
For production, make sure to restrict the source (CIDR or otherwise) to only the elements that need to access the DB.

```bash
aws ec2 create-security-group --group-name <security-group-name> --description "pgtest security group"
aws ec2 authorize-security-group-ingress --group-name <security-group-name> --protocol tcp --port 5432 --cidr 0.0.0.0/0
```

Fetch the Security Group ID
``` bash
$ aws ec2 describe-security-groups --filters "Name=group-name,Values=<security-group-name>" --query "SecurityGroups[*].GroupId"
```

Now create an RDS instance with the PostgreSQL engine

```bash
aws rds create-db-instance \
    --publicly-accessible \
    --allocated-storage 20 --db-instance-class db.t3.micro \
    --db-instance-identifier <instance-name> \
    --engine postgres \
    --engine-version 15.2
    --master-username <master-username> \
    --vpc-security-group-ids <security-group-id> \ # Sec group with TCP 5432 inbound rule
    --master-user-password <db-password>

aws rds wait db-instance-available --db-instance-identifier=<instance-name>
```
## Create configmap

Create a configmap that contains information to connect to the RDS DB instance

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: dbconfig
data:
  region: <region-name> #Region in which instance is to be created
  instance_name: <instance-name>  # name of the RDS Instance that would be provisioned
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
  postgres_username: <master-username>
  postgres_password: <db-password> 
```

**NOTE**

The AWS credentials must have proper permissions to perform operations on RDS.

## Installing the PostgreSQL Chart

To install the PostgreSQL chart with the release name `my-release` and default configuration, run the following commands:

```bash
$ helm repo add bitnami https://charts.bitnami.com/bitnami
$ helm repo update

$ helm install my-release --create-namespace --namespace postgres-test bitnami/postgresql
```

> **Tip**: List all releases using `helm list`


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

## Create Docker Image to be used in the Blueprint
Create the docker image which includes awscli, psql and kando (https://docs.kanister.io/tooling.html#kando) . This image will be provided in the Kanister blueprint that we are going to create later.

```bash
$ docker build -t <image-registry>/<image-repo>:<tag> .
$ docker push <image-registry>/<image-repo>:<tag>
```

### Create Blueprint

Modify the `rds-restore-blueprint.yaml` file and update the `image` field of the `Phases` to use the docker image built in the previous step.  

Create a Blueprint in the same namespace as the controller. 

```bash
$ kubectl create -f ./rds-restore-blueprint.yaml -n kanister
```

Once Postgres is running, you can populate it with some data. Let's add a table called "company" to a "test" database:

```bash
$ export POSTGRES_PASSWORD=$(kubectl get secret --namespace postgres-test my-release-postgresql -o jsonpath="{.data.postgres-password}" | base64 -d)

$ kubectl run my-release-postgresql-client --rm --tty -i --restart='Never' --namespace postgres-test --image docker.io/bitnami/postgresql:15.1.0-debian-11-r31 --env="PGPASSWORD=$POSTGRES_PASSWORD" \
      --command -- psql --host my-release-postgresql -U postgres -d postgres -p 5432

postgres=# 
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

You can now take a backup of the PostgresDB data by creating backup ActionSet for this application. Create an ActionSet in the same namespace as the controller.

```bash
$ kubectl get profile -n postgres-test
NAME               AGE
s3-profile-7d6wt   7m25s

$ kanctl create actionset --action backup --namespace kanister --blueprint rds-postgres-bp --statefulset postgres-test/my-release-postgresql --profile postgres-test/s3-profile-7d6wt
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

To verify, Connect to the RDS PostgreSQL database instance using the command mentioned below.

In this command `<instance-name>`, `<db-password>`, and `<master-username>` are the details of the RDS instance that we have already created as part of [setup](#create-rds-instance-on-aws)

```bash
$ kubectl run my-release-postgresql-client --rm --tty -i --restart='Never' --namespace postgres-test --image docker.io/bitnami/postgresql:15.1.0-debian-11-r31 --env="PGPASSWORD=<db-password>" --command -- psql --host <instance-name> -U <master-username> -d template1 -p 5432                   

psql (15.1, server 15.2)
SSL connection (protocol: TLSv1.2, cipher: ECDHE-RSA-AES256-GCM-SHA384, compression: off)
Type "help" for help.

postgres=> \l
                                                 List of databases
   Name    |  Owner   | Encoding |   Collate   |    Ctype    | ICU Locale | Locale Provider |   Access privileges   
-----------+----------+----------+-------------+-------------+------------+-----------------+-----------------------
 postgres  | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |            | libc            | 
 rdsadmin  | rdsadmin | UTF8     | en_US.UTF-8 | en_US.UTF-8 |            | libc            | rdsadmin=CTc/rdsadmin+
           |          |          |             |             |            |                 | rdstopmgr=Tc/rdsadmin
 template0 | rdsadmin | UTF8     | en_US.UTF-8 | en_US.UTF-8 |            | libc            | =c/rdsadmin          +
           |          |          |             |             |            |                 | rdsadmin=CTc/rdsadmin
 template1 | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |            | libc            | postgres=CTc/postgres+
           |          |          |             |             |            |                 | =c/postgres
 test      | postgres | UTF8     | en_US.UTF-8 | en_US.UTF-8 |            | libc            | 
(5 rows)

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
$ kanctl --namespace kanister create actionset --action delete --from backup-glptq --namespacetargets kanister
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