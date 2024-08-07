This example shows how you can backup and restore the application that is deployed using [DeploymentConfig](https://docs.OpenShift.com/container-platform/4.1/applications/deployments/what-deployments-are.html#deployments-and-deploymentconfigs_what-deployments-are),
[OpenShift](https://www.openshift.com/)'s resource on an OpenShift cluster. Like Deployment, Statefulset in Kubernetes, OpenShift has another controller named DeploymentConfig, it is almost like Deployment but has some significant differences.

# DeploymentConfig

[DeploymentConfig](https://docs.openshift.com/container-platform/4.1/applications/deployments/what-deployments-are.html#deployments-and-deploymentconfigs_what-deployments-are) is not standard
Kubernetes resource but [OpenShift](https://www.openshift.com/) resource and creates a new
ReplicationController and let's it start up Pods.

This example can be followed if your application is deployed on [OpenShift](https://www.openshift.com/)
cluster's DeploymentConfig resources.

## Prerequisites

- Setup OpenShift, you can follow steps mentioned below
- PV provisioner support in the underlying infrastructure
- Kanister controller version 0.110.0 installed in your cluster in namespace `kanister`
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)


# Setup OpenShift

To test the applications that are deployed on OpenShift using DeploymentConfig, we will first have to setup
OpenShift cluster. To setup the OpenShift cluster in our local environment we are going to use [minishift](https://github.com/minishift/minishift).
To install and setup minishift please follow [this guide](https://docs.okd.io/latest/minishift/getting-started/index.html).

Once we have minishift setup we will go ahead and deploy PostgreSQL application on the minishift cluster using DeploymentConfig and try to backup and restore the data from the application.

**Note**

Once you have setup minishift by following the steps mentioned above, you can interact with the cluster using `oc` command line tool. By default
you are logged in as developer user, that will prevent us from creating some of the resources so please make sure you login as admin by
the following command
```
oc login -u system:admin
```
and you are all set to go through rest of this example.

# Install PostgreSQL

We will use the JSON templates, as described [here](https://github.com/openshift/origin/tree/master/examples/db-templates), provided by OpenShift to deploy the PostgreSQL database on the minishift cluster that we have just setup.

To install the PostgreSQL on your cluster please run below command

```bash
# Create the namespaces
~ oc create ns postgres-test
namespace/postgres-test created

# Install the database
~ oc new-app https://raw.githubusercontent.com/openshift/origin/master/examples/db-templates/postgresql-persistent-template.json -n postgres-test \
    -e POSTGRESQL_ADMIN_PASSWORD=secretpassword
```

We can use the environment variable `POSTGRESQL_ADMIN_PASSWORD` to set the `ADMIN` password for our PostgreSQL installation. Once the database is installed, fetch all the pods from the namespace to make sure all the pods are in `Running` status.

```bash
~ oc get pods -n postgres-test
NAME                 READY     STATUS    RESTARTS   AGE
postgresql-1-72k8g   1/1       Running   0          2m
```

**Note**
The secret that gets created after installation of PostgreSQL doesn't have the ADMIN password that we have just specified and this password gets used by the blueprint to connect to the PostgreSQL instance and perform the activities.
To address above issue, we will have to manually create a secret that will have this ADMIN password for the key `postgresql_admin_password`. Please use below command to create the secret

```bash
~ oc create secret generic postgresql-postgres-test -n postgres-test  --from-literal=postgresql_admin_password=secretpassword
secret/postgresql-postgres-test created
```

## Integrating with Kanister

When we say integrating with Kanister, we actually mean creating some Kanister resources to support `backup` and `restore`
actions.

### Create Profile

Create Profile Kanister resource using below command

```bash
~ kanctl create profile s3compliant --access-key <ACCESS-KEY> \
        --secret-key <SECRET-KEY> \
        --bucket <BUKET-NAME> --region <AWS-REGION> \
        --namespace postgres-test
secret 's3-secret-htzs2i' created
profile 's3-profile-2rzmc' created
```

**NOTE:**

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.

### Create Blueprint

**NOTE: v2 Blueprints are experimental and are not supported with standalone Kanister.**

Create Blueprint in the same namespace as the controller (`kanister`)

```
~ oc create -f postgres-dep-config-blueprint.yaml -n kanister
blueprint.cr.kanister.io/postgres-bp created
```

Now that we have created the Profile and Blueprint Kanister resources we will insert some data into
PostgreSQL database that we will take backup of.
To insert the data into the PostgreSQL database we will `exec` into the PostgreSQL pod, please follow below commands
to do so

```bash
# Get the PostgreSQL pod
~ oc get pods -n postgres-test
NAME                 READY     STATUS    RESTARTS   AGE
postgresql-1-72k8g   1/1       Running   0          17m

# Exec into the PostgreSQL pod
~ oc exec -it -n postgres-test postgresql-1-72k8g bash

## use psql cli to add entries in postgresql database

bash-4.2$ PGPASSWORD=${POSTGRESQL_ADMIN_PASSWORD} psql -U ${PGUSER}
psql (10.6)
Type "help" for help.

postgres=# CREATE DATABASE test;

postgres=# \l
                                 List of databases
   Name    |  Owner   | Encoding |  Collate   |   Ctype    |   Access privileges
-----------+----------+----------+------------+------------+-----------------------
 postgres  | postgres | UTF8     | en_US.utf8 | en_US.utf8 |
 sampledb  | userC3S  | UTF8     | en_US.utf8 | en_US.utf8 |
 template0 | postgres | UTF8     | en_US.utf8 | en_US.utf8 | =c/postgres          +
           |          |          |            |            | postgres=CTc/postgres
 template1 | postgres | UTF8     | en_US.utf8 | en_US.utf8 | =c/postgres          +
           |          |          |            |            | postgres=CTc/postgres
 test      | postgres | UTF8     | en_US.utf8 | en_US.utf8 |
(5 rows)

# Connect to test database1
postgres=# \c test
You are now connected to database "test" as user "postgres".

# Create company table in the test database
test=# CREATE TABLE COMPANY(
test(# ID INT PRIMARY KEY     NOT NULL,
test(# NAME           TEXT    NOT NULL,
test(# AGE            INT     NOT NULL,
test(# ADDRESS        CHAR(50),
test(# SALARY         REAL,
test(# CREATED_AT    TIMESTAMP
test(# );
CREATE TABLE

# Insert some data into the table
test=# INSERT INTO COMPANY (ID,NAME,AGE,ADDRESS,SALARY,CREATED_AT) VALUES (10, 'Paul', 32, 'California', 20000.00, now());
INSERT 0 1

test=# INSERT INTO COMPANY (ID,NAME,AGE,ADDRESS,SALARY,CREATED_AT) VALUES (11, 'John', 31, 'NY', 20100.00, now());
INSERT 0 1

test=# INSERT INTO COMPANY (ID,NAME,AGE,ADDRESS,SALARY,CREATED_AT) VALUES (12, 'Robert', 28, 'Wash. DC', 21100.00, now());
INSERT 0 1

# list all the data from the company table in the test database
test=# select * from company;
 id |  name  | age |                      address                       | salary |         created_at
----+--------+-----+----------------------------------------------------+--------+----------------------------
 10 | Paul   |  32 | California                                         |  20000 | 2020-03-02 12:03:28.919661
 11 | John   |  31 | NY                                                 |  20100 | 2020-03-02 12:04:03.766175
 12 | Robert |  28 | Wash. DC                                           |  21100 | 2020-03-02 12:05:18.345847
(3 rows)

```

## Protect the Application

Now that we have inserted some data into the database let's try to create Actionset Kanister resource to take `backup`
of the data that we have inserted. Create an Actionset in the same namespace where your Kanister controller is

```bash
# get the name of the profile resource
~ oc get profile -n postgres-test
NAME               AGE
s3-profile-2rzmc   33m

# Create backup Actionset
~ kanctl create actionset --action backup --namespace kanister --blueprint postgres-bp  \
    --profile postgres-test/s3-profile-2rzmc \
    --deploymentconfig postgres-test/postgresql
actionset backup-tnf2d created

# check the status of the Actionset
~ oc describe actionset -n kanister backup-tnf2d
```

Once we have made sure that the backup action is complete, we can go ahead and delete the data from the database to imitate the disaster.

### Disaster strikes!
Let's say someone accidentally deleted the `test` database using the following command:

```bash
# Get the PostgreSQL pod name
~ oc get pods -n postgres-test
NAME                 READY     STATUS    RESTARTS   AGE
postgresql-1-72k8g   1/1       Running   0          56m

# Exec into the POD
~ oc exec -it -n postgres-test postgresql-1-72k8g bash

bash-4.2$ PGPASSWORD=${POSTGRESQL_ADMIN_PASSWORD} psql -U ${PGUSER}
psql (10.6)
Type "help" for help.

# list all the databases
postgres=# \l
                                List of databases
   Name    |  Owner   | Encoding |  Collate   |   Ctype    |   Access privileges
-----------+----------+----------+------------+------------+-----------------------
 postgres  | postgres | UTF8     | en_US.utf8 | en_US.utf8 |
 sampledb  | userC3S  | UTF8     | en_US.utf8 | en_US.utf8 |
 template0 | postgres | UTF8     | en_US.utf8 | en_US.utf8 | =c/postgres          +
           |          |          |            |            | postgres=CTc/postgres
 template1 | postgres | UTF8     | en_US.utf8 | en_US.utf8 | =c/postgres          +
           |          |          |            |            | postgres=CTc/postgres
 test      | postgres | UTF8     | en_US.utf8 | en_US.utf8 |
(5 rows)


# Drop the test database
postgres=# drop database test ;
DROP DATABASE

postgres=# \l
                                 List of databases
   Name    |  Owner   | Encoding |  Collate   |   Ctype    |   Access privileges
-----------+----------+----------+------------+------------+-----------------------
 postgres  | postgres | UTF8     | en_US.utf8 | en_US.utf8 |
 sampledb  | userC3S  | UTF8     | en_US.utf8 | en_US.utf8 |
 template0 | postgres | UTF8     | en_US.utf8 | en_US.utf8 | =c/postgres          +
           |          |          |            |            | postgres=CTc/postgres
 template1 | postgres | UTF8     | en_US.utf8 | en_US.utf8 | =c/postgres          +
           |          |          |            |            | postgres=CTc/postgres
(4 rows)


```
After dropping the `test` database as you can see we are only able to see 4 databases.

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
# Get the backup Actionset name
~ oc get actionset -n kanister
NAME           AGE
backup-tnf2d   7m

# Create restore Actionset
~ kanctl --namespace kanister create actionset --action restore --from backup-tnf2d
actionset restore-backup-tnf2d-jhnt7 created

# Check the status of the restore Actionset to make sure the actionset is completed
~ oc describe actionset -n kanister restore-backup-tnf2d-jhnt7

# and you will be able to see, in events section, that the Actionset is completed.
```

Once the status of the Actionset is completed, exec into the PostgreSQL pod once again and make sure the `test` database and all of it's records have
been restored successfully.

```bash
~ oc exec -it -n postgres-test postgresql-1-72k8g bash

bash-4.2$ PGPASSWORD=${POSTGRESQL_ADMIN_PASSWORD} psql -U ${PGUSER}
psql (10.6)
Type "help" for help.

# list all the databases
postgres=# \l
                                 List of databases
   Name    |  Owner   | Encoding |  Collate   |   Ctype    |   Access privileges
-----------+----------+----------+------------+------------+-----------------------
 postgres  | postgres | UTF8     | en_US.utf8 | en_US.utf8 |
 sampledb  | userC3S  | UTF8     | en_US.utf8 | en_US.utf8 |
 template0 | postgres | UTF8     | en_US.utf8 | en_US.utf8 | =c/postgres          +
           |          |          |            |            | postgres=CTc/postgres
 template1 | postgres | UTF8     | en_US.utf8 | en_US.utf8 | postgres=CTc/postgres+
           |          |          |            |            | =c/postgres
 test      | postgres | UTF8     | en_US.utf8 | en_US.utf8 |
(5 rows)

# As you can see we have restored the test database
# connect to the test database

postgres=# \c test
You are now connected to database "test" as user "postgres".
test=# select * from company;
 id |  name  | age |                      address                       | salary |         created_at
----+--------+-----+----------------------------------------------------+--------+----------------------------
 10 | Paul   |  32 | California                                         |  20000 | 2020-03-02 12:03:28.919661
 11 | John   |  31 | NY                                                 |  20100 | 2020-03-02 12:04:03.766175
 12 | Robert |  28 | Wash. DC                                           |  21100 | 2020-03-02 12:05:18.345847
(3 rows)

```
As you can see we have successfully restored the data into the PostgreSQL database using the backup that we had already created.

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
~  kanctl --namespace kanister  create actionset --action delete --from backup-tnf2d
actionset delete-backup-tnf2d-qxl6k created

# View the status of the ActionSet
$ kubectl --namespace kanister describe actionset delete-backup-tnf2d-qxl6k
```

## Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset <actionset-name> -n kanister
```

## Cleanup

### Uninstalling the application

When we install a application using `oc new-app` a label of format `app=<name>`, gets added to all the resources that get created. To get that label execute below command

```
~ oc get pods -n postgres-test --show-labels
NAME                 READY     STATUS    RESTARTS   AGE       LABELS
postgresql-1-72k8g   1/1       Running   0          1h        app=postgresql-persistent,deployment=postgresql-1,deploymentconfig=postgresql,name=postgresql
```

As you can see the label that was added is `app=postgresql-persistent` to delete all the resources with this label, use below command

```bash
~  oc delete all -n postgres-test -l app=postgresql-persistent
```



### Delete CRs
Remove Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io <blueprint-name> -n kanister

$ kubectl delete profiles.cr.kanister.io <profile-name> -n postgres-test
```
