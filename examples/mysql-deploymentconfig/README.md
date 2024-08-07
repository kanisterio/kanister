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

**Note**

While installing Kanister using helm please make sure that you have `kanister` namespace already created, using below command
```bash
~ oc create ns kanister
```
and then you can just follow the steps that are mentioned in [official docs](https://docs.kanister.io/install.html#id2) to install Kanister.

# Setup OpenShift

To test the applications that are deployed on OpenShift using DeploymentConfig, we will first have to setup
OpenShift cluster. To setup the OpenShift cluster in our local environment we are going to use [minishift](https://github.com/minishift/minishift).
To install and setup minishift please follow [this guide](https://docs.okd.io/latest/minishift/getting-started/index.html).

Once we have minishift setup we will go ahead and deploy MySQL application on the minishift cluster using DeploymentConfig and try to backup and restore the data from the application.

**Note**

Once you have setup minishift by following the steps mentioned above, you can interact with the cluster using `oc` command line tool. By default
you are logged in as developer user, that will prevent us from creating some of the resources so please make sure you login as admin by
following below command
```
oc login -u system:admin
```
and you are all set to go through rest of this example.

It is not necessary to use minishift to go through this example, you can go through this example as long as you have an OpenShift cluster setup.

# Install MySQL

We will follow the steps that are suggested by OpenShift and can be found [here](https://github.com/openshift/origin/tree/master/examples/db-templates). Once you have the minishift setup, please go ahead and run below commands to deploy the
MySQL application

```bash
~ oc create ns mysql-test
~ oc new-app https://raw.githubusercontent.com/openshift/origin/master/examples/db-templates/mysql-ephemeral-template.json \
        -n mysql-test \
        -p MYSQL_ROOT_PASSWORD=secretpassword
```


Once you have deployed the MySQL application, please verify the status of the MySQL pod using below command to make sure the pod is in `running` status

```bash
~ oc get pods -n mysql-test
NAME                       READY     STATUS    RESTARTS   AGE
mysql-1-bzgcr              1/1       Running   0          27s
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
        --namespace mysql-test

secret 's3-secret-8w3goj' created
profile 's3-profile-44xjl' created
```

**NOTE:**

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.

### Create Blueprint

**NOTE: v2 Blueprints are experimental and are not supported with standalone Kanister.**

Create Blueprint in the same namespace as the controller

```bash
~ oc create -f mysql-dep-config-blueprint.yaml -n kanister

blueprint.cr.kanister.io/mysql-dep-config-blueprint created
```

Now that we have created the Profile and Blueprint Kanister resources we will insert some data into
MySQL database that we will take backup of.
To insert the data into the MySQL database we will `exec` into the MySQL pod, please follow below commands
to do so

```bash
~ oc get pods -n mysql-test
NAME                       READY     STATUS    RESTARTS   AGE
mysql-1-bzgcr              1/1       Running   0          13m

# Exec into the pod and insert some data
~ oc exec -it -n mysql-test mysql-1-bzgcr bash
bash-4.2$  mysql -u root
Welcome to the MySQL monitor.  Commands end with ; or \g.
Your MySQL connection id is 1
Server version: 5.6.24 MySQL Community Server (GPL)

Copyright (c) 2000, 2015, Oracle and/or its affiliates. All rights reserved.

Oracle is a registered trademark of Oracle Corporation and/or its
affiliates. Other names may be trademarks of their respective
owners.

Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

mysql> show databases;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| mysql              |
| performance_schema |
| testdb             |
+--------------------+
4 rows in set (0.01 sec)

# create a new database and insert some records
mysql> create database company;
Query OK, 1 row affected (0.00 sec)

mysql> use company;
Database changed

mysql> create table employees (name varchar(100), dept varchar(100), age int);
Query OK, 0 rows affected (0.02 sec)

mysql> insert into employees values ("Tom", "Engg.", 25);
Query OK, 1 row affected (0.01 sec)

mysql> insert into employees values ("Vivek", "Engg.", 20);
Query OK, 1 row affected (0.01 sec)

# Once we have inserted the data lets try to get all the data
mysql> select * from employees;
+-------+-------+------+
| name  | dept  | age  |
+-------+-------+------+
| Tom   | Engg. |   25 |
| Vivek | Engg. |   20 |
+-------+-------+------+
2 rows in set (0.00 sec)


```


## Protect the Application

Now that we have inserted some data into the database let's try to create Actionset Kanister resource to take `backup`
of the data that we have inserted. Create an Actionset in the same namespace where your Kanister controller is

```bash
# get the name of the profile resource
~ oc get profile -n mysql-test
NAME               AGE
s3-profile-44xjl   16m

# create backup actionset
# please make note here that the value of the --deploymentconfig flag
# is the name of the deploymentconfig through which the MySQL instance is running
~ kanctl create actionset --action backup --namespace kanister --blueprint mysql-dep-config-blueprint  \
    --profile mysql-test/s3-profile-44xjl \
    --deploymentconfig mysql-test/mysql-56-centos7 \
    --secrets mysql=mysql-test/mysql
actionset backup-vmgsp created

# you can describe the actionset to make sure the actionset is completed
~ oc describe actionset backup-vmgsp -n kanister
```

### Disaster strikes!

Let's say someone accidentally deleted the databases that we have created using below commands

```bash
~ oc get pods -n mysql-test
NAME                       READY     STATUS    RESTARTS   AGE
mysql-1-bzgcr              1/1       Running   0          13m

# Exec into the pod and insert some data
~ oc exec -it -n mysql-test mysql-1-bzgcr bash
bash-4.2$  mysql -u root
Welcome to the MySQL monitor.  Commands end with ; or \g.
Your MySQL connection id is 1
Server version: 5.6.24 MySQL Community Server (GPL)

Copyright (c) 2000, 2015, Oracle and/or its affiliates. All rights reserved.

Oracle is a registered trademark of Oracle Corporation and/or its
affiliates. Other names may be trademarks of their respective
owners.

Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

mysql> drop database testdb;
mysql> drop database company;
```
Once you have deleted the database, you can list all the databases once again to make sure
the databases has been deleted.

```bash
mysql> show databases;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| mysql              |
| performance_schema |
+--------------------+
3 rows in set (0.00 sec)

```

### Restore the Application

To restore the data/databases that we have deleted in previous step to imitate disaster, we will use the backup
that we have already created and create an Actionset with action as `restore`. Please follow below  command to
create the Actionset

```bash
~ kanctl --namespace kanister  create actionset --action restore --from "backup-vmgsp"
actionset restore-backup-vmgsp-6shbk created

# make sure the status of the actionset is completed to verify that the restore is complete
~ oc describe actionset restore-backup-vmgsp-6shbk -n kanister
```

Once you are sure that the restore actionset is complete you can login again into the MySQL database to check
if the data has been restored successfully.

```bash
mysql> show databases;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| company            |
| mysql              |
| performance_schema |
| testdb             |
+--------------------+
5 rows in set (0.00 sec)


mysql> select * from company.employees;
+-------+-------+------+
| name  | dept  | age  |
+-------+-------+------+
| Tom   | Engg. |   25 |
| Vivek | Engg. |   20 |
+-------+-------+------+
2 rows in set (0.02 sec)

```
As you can see the databases as well as the data that was inside those database is successfully restored.


## Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
~ oc --namespace kanister logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
~ oc describe actionset <actionset-name> -n kanister
```


## Cleanup

### Uninstalling the application

To uninstall/delete the MySQL application:

```bash
~ oc delete <resource-name>
```

### Delete Kanister resources
Remove Blueprint and Profile CR

```bash
~ oc delete blueprints.cr.kanister.io <blueprint-name> -n kanister

~ oc delete profiles.cr.kanister.io <profile-name> -n mysql-test
```