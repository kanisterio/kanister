# CockroachDB

[CockroachDB](https://www.cockroachlabs.com/) is one of the database servers based on SQL.

## Introduction

This chart bootstraps a three node CockroachDB deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.16+ with Beta APIs enabled
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.80.0 installed in your cluster, let's assume in Namespace `kanister`
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)

## Installing the Chart

The commands mentioned below could be run to install the CockroachDB database using the `cockroachdb` chart with the release name `cockroachdb-release`:

```bash
# Add cockroachdb in your local chart repository
$ helm repo add cockroachdb https://charts.cockroachdb.com/

# Update your local chart repository
$ helm repo update

# Install the cockroachdb in cockroachdb namespace
$ helm install cockroachdb-release --namespace cockroachdb --create-namespace cockroachdb/cockroachdb

# In order to interact with the CockroachDB Cluster, install the secure cockroachdb client
# Download latest secure client manifest 
$ curl -O https://raw.githubusercontent.com/cockroachdb/helm-charts/master/examples/client-secure.yaml

# Edit Fields in client-secure.yaml. A sample manifest file is being included for reference.
# Value of spec.serviceAccountName should be same as your release name
spec.serviceAccountName: cockroachdb-release
# Find the latest tag of image cockroachdb/cockroach from https://github.com/cockroachdb/helm-charts/blob/master/cockroachdb/values.yaml
spec.image: cockroachdb/cockroach:<VERSION>
# Value of spec.volumes[0].project.sources[0].secret.name should be <release-name>-client-secret
spec.volumes[0].project.sources[0].secret.name: cockroachdb-release-client-secret

$ kubectl create -f ./client-secure.yaml -n cockroachdb
```

> **Tip**: List all releases using `helm list --all-namespaces`, using Helm Version 3.

## Integrating with Kanister

If you have deployed CockroachDB application with name other than `cockroachdb-release` and namespace other than `cockroachdb`, you need to modify the commands (backup, restore and delete) used below to use the correct release name and namespace.
### Create Profile
Create Profile if not created already, and set the values accordingly

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--bucket <s3-bucket-name> --region <region-name> \
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

Create Blueprint in the same namespace as the Kanister controller

```bash
$ kubectl create -f ./cockroachdb-blueprint.yaml -n kanister
```

Once CockroachDB is running, you can populate it with some data. Let's add a table called "accounts" to a bank database:

```bash
# Connect to Secure Client Shell by running a shell inside cockroachdb-client-secure pod
$ kubectl exec -it -n cockroachdb cockroachdb-client-secure -- ./cockroach sql --certs-dir=./cockroach-certs --host=cockroachdb-release-public

# From inside the shell, use the CLI to insert some data into the bank database
# Create "bank" db
> CREATE DATABASE bank;

# Create "accounts" table
> CREATE TABLE bank.accounts (id INT PRIMARY KEY, balance DECIMAL);

# Insert row to the table
> INSERT INTO bank.accounts VALUES (1, 1000.50);
> INSERT INTO bank.accounts VALUES (2, 2000.70);

# View data in "accounts" table
> SELECT * FROM bank.accounts;

  id | balance
+----+---------+
   1 | 1000.50
   2 | 2000.70
(2 rows)

```

## Protect the Application

You can now take a backup of the CockroachDB data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
# Find profile name
$ kubectl get profile -n kanister
NAME               AGE
s3-profile-drnw9   2m

# Find client secret name
$ kubectl get secrets -n cockroachdb
NAME                                        TYPE                 DATA   AGE
cockroachdb-release-ca-secret               Opaque               2      6h38m
cockroachdb-release-client-secret           kubernetes.io/tls    3      6h38m       <<---
cockroachdb-release-node-secret             kubernetes.io/tls    3      6h38m
sh.helm.release.v1.cockroachdb-release.v1   helm.sh/release.v1   1      6h38m

# Create Actionset
# Please make sure the value of blueprint matches with the name of blueprint that we have created already
$ kanctl create actionset --action backup --namespace kanister --blueprint cockroachdb-blueprint --statefulset cockroachdb/cockroachdb-release --profile kanister/s3-profile-drnw9 --secrets cockroachSecret=cockroachdb/cockroachdb-release-client-secret
actionset backup-rslmb created

$ kubectl --namespace kanister get actionsets.cr.kanister.io
NAME                 AGE
backup-rslmb         1m

# View the status of the actionset
# Please make sure the name of the actionset here matches with name of the name of actionset that we have created already
$ kubectl --namespace kanister describe actionset backup-rslmb
```

### Disaster strikes!

Let's say someone accidentally deleted the test database using the following command:
```bash
# Connect to Secure Client Shell by running a shell inside cockroachdb-client-secure pod
$ kubectl exec -it -n cockroachdb cockroachdb-client-secure -- ./cockroach sql --certs-dir=./cockroach-certs --host=cockroachdb-release-public

# Drop the test database
> SHOW DATABASES;
  database_name | owner | primary_region | regions | survival_goal
----------------+-------+----------------+---------+----------------
  bank          | root  | NULL           | {}      | NULL
  defaultdb     | root  | NULL           | {}      | NULL
  postgres      | root  | NULL           | {}      | NULL
  system        | node  | NULL           | {}      | NULL
(4 rows)

> DROP DATABASE bank CASCADE;
Query OK, 1 row affected (0.03 sec)

> SHOW DATABASES;
  database_name | owner | primary_region | regions | survival_goal
----------------+-------+----------------+---------+----------------
  defaultdb     | root  | NULL           | {}      | NULL
  postgres      | root  | NULL           | {}      | NULL
  system        | node  | NULL           | {}      | NULL
(3 rows)

```

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:
> **Note** : CockroachDB uses [Garbage Collection](https://www.cockroachlabs.com/docs/stable/architecture/storage-layer.html#garbage-collection), So in order to perform Full Cluster Restore on same cluster, one must wait out the Default Garbage Collection Run After Time i.e. 90,000 Seconds or 25 Hours.
> On the other hand the following step is Not Required if snapshot is being restored on a new DB cluster.
> In Order to reduce the Garbage Collection time, run the following command, wait for a minute and let cockroachdb to automatic garbage collection. And then use kanctl to create restore action.
```bash
$ ALTER RANGE default CONFIGURE ZONE USING gc.ttlseconds = 60;
```
```bash
# Make sure to use correct backup actionset name here
$ kanctl --namespace kanister create actionset --action restore --from "backup-rslmb"
actionset restore-backup-62vxm-2hdsz created

# View the status of the ActionSet
# Make sure to use correct restore actionset name here
$ kubectl --namespace kanister describe actionset restore-backup-62vxm-2hdsz
```

Once the ActionSet status is set to "complete", you can see that the data has been successfully restored to CockroachDB

```bash
> SHOW DATABASES;
  database_name | owner | primary_region | regions | survival_goal
----------------+-------+----------------+---------+----------------
  bank          | root  | NULL           | {}      | NULL
  defaultdb     | root  | NULL           | {}      | NULL
  postgres      | root  | NULL           | {}      | NULL
  system        | node  | NULL           | {}      | NULL
(4 rows)

> USE bank;

Database changed
> SHOW TABLES;
+----------------+
| Tables         |
+----------------+
| accounts       |
+----------------+
1 row in set (0.00 sec)

> SELECT * FROM bank.accounts;
  id | balance
+----+---------+
   1 | 1000.50
   2 | 2000.70
```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from backup-rslmb --namespacetargets kanister
actionset delete-backup-glptq-cq6bw created

# View the status of the ActionSet
$ kubectl --namespace kanister describe actionset delete-backup-glptq-cq6bw
```


## Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-62vxm-2hdsz -n kanister
```


## Cleanup

### Uninstalling the Chart

To uninstall/delete the `cockroachdb-release` deployment:

```bash
# Helm Version 3
$ helm delete cockroachdb-release -n cockroachdb
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

### Delete CRs
Remove Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io cockroachdb-blueprint -n kanister

$ kubectl get profiles.cr.kanister.io -n kanister
NAME               AGE
s3-profile-drnw9   122m

$ kubectl delete profiles.cr.kanister.io s3-profile-drnw9 -n kanister
```
