# PostgreSQL-HA

[PostgreSQL](https://www.postgresql.org/) is an object-relational database management system (ORDBMS) with an emphasis on extensibility and on standards-compliance. 

## Introduction

This Helm chart has been developed based on [PostgreSQL](https://github.com/bitnami/charts/tree/master/bitnami/postgresql) chart but including some changes to guarantee high availability such as:

- A new deployment, service have been added to deploy [Pgpool-II](https://pgpool.net/mediawiki/index.php/Main_Page) to act as proxy for PostgreSQL backend. It helps to reduce connection overhead, acts as a load balancer for PostgreSQL, and ensures database node failover.
- Replacing `bitnami/postgresql` with `bitnami/postgresql-repmgr` which includes and configures repmgr. Repmgr ensures standby nodes assume the primary role when the primary node is unhealthy.

## Requirements
When restoring the postgreSQL with high availability in a different namespace, the standby instance pod goes into `CrashLoopBackOff` since the connection info for the primary/secondary nodes in the `repmgr` database points to source namespace. The blueprint `postgres-ha-hook.yaml` can be used to solve this issue which will update the `repmgr` database with correct connection information for primary and secondary instances

**NOTE:**

This blueprint is only required when you face above mentioned issue, else you will only be installing the helm chart for postgreSQL with high availability by following the steps.

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

$ kubectl create ns postgres-ha-test
$ helm install my-release --namespace postgres-ha-test bitnami/postgresql-ha
```

The command deploys PostgreSQL HA on the Kubernetes cluster in the default configuration.

> **Tip**: List all releases using `helm list`

```

### Create Blueprint
Create Blueprint in the same namespace as the controller

```bash
$ kubectl create -f ./postgres-ha-hook-blueprint.yaml -n kanister
```

### After Restore of the application
Let's assume, restore is done in a namespace `postgres-ha-test2`

The pod `my-release-postgresql-ha-postgresql-1` in `postgres-ha-test2` goes into `CrashLoopBackoff` because of wrong connection information.

Check the primary/standby connection information in rpmgr database 

```
Log into the primary node of postgres-ha application and get shell access
$ kubectl exec -ti my-release-postgresql-ha-postgresql-0 -n postgres-ha-test2 -- bash

I have no name!@my-release-postgresql-ha-postgresql-0:/$ PGPASSWORD=${POSTGRES_PASSWORD} psql -U $POSTGRES_USER
psql (11.14)
Type "help" for help.

postgres=# \c repmgr
You are now connected to database "repmgr" as user "postgres".

repmgr=# select conninfo from repmgr.nodes;
                                                                                               conninfo                                       
                                                         
----------------------------------------------------------------------------------------------------------------------------------------------
---------------------------------------------------------
 user=repmgr password=ktA5BvUN4a host=my-release-postgresql-ha-postgresql-1.my-release-postgresql-ha-postgresql-headless.postgres-ha-test.svc.
cluster.local dbname=repmgr port=5432 connect_timeout=5
 user=repmgr password=ktA5BvUN4a host= my-release-postgresql-ha-postgresql-0.my-release-postgresql-ha-postgresql-headless.postgres-ha-test.svc
.cluster.local dbname=repmgr port=5432 connect_timeout=5

```

The `conninfo` column is pointing to source namespace `postgres-ha-test`. Hence we need to create an actionset on statefulset in namespace `postgres-ha-test2` as that will update the `conninfo` column for primary/standy nodes with correct information.  An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
$ kanctl create actionset --action postrestorehook --namespace kanister --blueprint postgresql-hooks --statefulset postgres-ha-test2/my-release-postgresql-ha-postgresql
actionset postrestorehook-z2x49 created


## Check status
$ kubectl describe actionsets.cr.kanister.io postrestorehook-z2x49 --namespace kanister
```

Once the ActionSet status is set to "complete", The secondary instance pod `my-release-postgresql-ha-postgresql-1` starts successfully.

Verify the connection information by logging into primary instance `my-release-postgresql-ha-postgresql-0`

```
I have no name!@my-release-postgresql-ha-postgresql-0:/$ PGPASSWORD=${POSTGRES_PASSWORD} psql -U $POSTGRES_USER
psql (11.14)
Type "help" for help.

postgres=# \c repmgr
You are now connected to database "repmgr" as user "postgres".

repmgr=# select conninfo from repmgr.nodes;
                                                                                               conninfo                                       
                                                         
----------------------------------------------------------------------------------------------------------------------------------------------
---------------------------------------------------------
 user=repmgr password=ktA5BvUN4a host=my-release-postgresql-ha-postgresql-1.my-release-postgresql-ha-postgresql-headless.postgres-ha-test2.svc.
cluster.local dbname=repmgr port=5432 connect_timeout=5
 user=repmgr password=ktA5BvUN4a host= my-release-postgresql-ha-postgresql-0.my-release-postgresql-ha-postgresql-headless.postgres-ha-test2.svc
.cluster.local dbname=repmgr port=5432 connect_timeout=5

```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kubectl delete actionsets.cr.kanister.io postrestorehook-z2x49 -n kanister
actionset.cr.kanister.io "postrestorehook-z2x49" deleted

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

### Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release -n postgres-ha-test

Delete the namespace where the application is restored
$ kubectl delete -n postgres-ha-test2
```

### Delete CRs
Remove Blueprint 

```bash
$ kubectl delete blueprints.cr.kanister.io postgresql-hooks -n kanister 
blueprint.cr.kanister.io "postgresql-hooks" deleted

```
