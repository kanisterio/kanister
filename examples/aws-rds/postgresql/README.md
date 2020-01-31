# AWS RDS PostgreSQL engine

[PostgreSQL](https://www.postgresql.org/) is an object-relational database management system (ORDBMS) with an emphasis on extensibility and on standards-compliance.

## Introduction

This example is to demonstrate how Kanister can be integrated with AWS RDS instance to protect your data using Kanister functions

## Prerequisites

- Kubernetes 1.10+
- Kanister controller version 0.25.0 installed in your cluster
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## Create RDS instance on AWS

> You can skip this step if you already have an RDS instance created

RDS instance needs to be reachable from outside world. So make sure that you have VPC with security group having rule to allow ingress traffic on 5432 TCP port.


You can create security group and add rules to defaul VPC using following commands

```bash
aws ec2 create-security-group --group-name pgtest-sg --description "pgtest security group"
aws ec2 authorize-security-group-ingress --group-name pgtest-sg --protocol tcp --port 5432 --cidr 0.0.0.0/0
```

Now create a RDS instance with postgresql engine

```bash
aws rds create-db-instance \
    --publicly-accessible \
    --allocated-storage 20 --db-instance-class db.t2.micro	 \
    --db-instance-identifier test-postgresql-instance \
    --engine postgres \
    --master-username master \
    --vpc-security-group-id sg-xxxxyyyyzzz \ # Sec group with TCP 5432 inbound rule
    --master-user-password secret99

aws rds wait db-instance-available --db-instance-identifier=test-postgresql-instance
```

## Create configmap

Create a configmap which contains information to connect to the RDS DB instance

```
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    kasten.io/config: dataservice
  name: dbconfig
data:
  postgres.instanceid: test-rds-postgresql
  postgres.host: test-rds-postgresql.example.ap-south-1.rds.amazonaws.com
  postgres.databases: |
    - postgres
    - template1
  postgres.secret: dbcreds # name of K8s secret in the same namespace
```

## Integrating with Kanister

### Create Profile

Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--region <region-name> \
	--namespace pgtestrds
```

**NOTE:**

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.


### Create Blueprint

There are two ways that you can use to backup and restore RDS instance data:


1. Create RDS instance snapshot - Using `rds-postgres-snap-blueprint.yaml` Blueprint
2. Create RDS snapshot, extract postgres data and push that data to S3 storage - Using `rds-postgres-dump-blueprint.yaml` Blueprint 

So as you can see we will have to create a blueprint depending on how are we going to take the backup.

Use `rds-postgres-snap-blueprint.yaml` Blueprint if you want to take backup using RDS snapshots or you can use `rds-postgres-dump-blueprint.yaml` Blueprint if you want to extract postgres dump from snapshot and push to S3 storage


```bash
$ kubectl create -f <blueprint> -n kasten-io
```

## Protect the Application

You can now take a snapshot of the PostgreSQL RDS instance data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

> If you have deployed your application which uses RDS instance in namespace other than `pgtestrds`, you need to modify the commands used below to use the correct namespace

```bash
$ kubectl get profile -n pgtestrds
NAME               AGE
s3-profile-sph7s   2h


# Use correct blueprint name (one of `rds-postgres-dump-bp` or `rds-postgres-snapshot-bp`) you have created earlier
# 
$ kanctl create actionset --action backup --namespacetargets pgtestrds --config-maps dbconfig=pgtestrds/dbconfig --profile pgtestrds/s3-profile-6hmhn -b <blueprint-name> -n kasten-io
actionset backup-llfb8 created

# Where, 
# dbconfig is a configmap holding RDS infromation
# Please see pgtest/deploy/config.yaml for configmap format

$ kubectl --namespace kasten-io get actionsets.cr.kanister.io
NAME                 AGE
backup-llfb8         2h

# View the status of the actionset
$ kubectl --namespace kasten-io describe actionset backup-llfb8
```

### Restore the Application

To restore the missing data from RDS snapshot, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets


```bash
$ kanctl create actionset --namespace kasten-io --action restore --config-maps dbconfig=pgtestrds/dbconfig --from backup-llfb8
actionset restore-backup-llfb8-64gqm created

## Check status
$ kubectl --namespace kasten-io describe actionset restore-backup-llfb8-64gqm
```


### Delete snapshot

The snapshot created by Actionset can be deleted by the following command

```bash
$ kanctl create actionset --namespace kasten-io --action delete -c dbconfig=pgtestrds/dbconfig --from backup-llfb8
actionset "delete-backup-llfb8-k9ncm" created

## Check status
$ kubectl --namespace kasten-io describe actionset delete-backup-llfb8-k9ncm

```

## Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kasten-io logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-md6gb-d7g7w -n kasten-io
```

## Cleanup

```console
$ kubectl delete -f <blueprint-name> -n kasten-io
```

### Delete CRs
Remove Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io <blueprint-name> -n kasten-io

$ kubectl get profiles.cr.kanister.io -n pgtestrds
NAME               AGE
s3-profile-zvrg9   125m
$ kubectl delete profiles.cr.kanister.io s3-profile-zvrg9 -n pgtestrds
```
