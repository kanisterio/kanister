
## Installation

Follow this document to install the operator https://access.crunchydata.com/documentation/postgres-operator/latest/quickstart

```
cd postgres-operator-examples

kubectl apply -k kustomize/install/namespace

kubectl apply --server-side -k kustomize/install/default
```


Check the status of the operator

```bash
kubectl -n postgres-operator get pods \
  --selector=postgres-operator.crunchydata.com/control-plane=postgres-operator \
  --field-selector=status.phase=Running
NAME                   READY   STATUS    RESTARTS   AGE
pgo-6cc745c948-4csn9   1/1     Running   0          48m
```


## create a postgres cluster

```bash
kubectl apply -k kustomize/postgres
```

This will create a postgres cluster `hippo` in `postgres-operator` namespace.

Check the cluster prgoress

```bash
kubectl -n postgres-operator describe postgresclusters.postgres-operator.crunchydata.com hippo
```

## Install Kanister

```bash
helm install kanister -n kanister kanister/kanister-operator --create-namespace
```

### create blueprint

```bash
~/work/opensource/kanister/examples/pgo (pgo-blueprint*) » k create -f pgo-blueprint.yaml -n kanister
blueprint.cr.kanister.io/pgo-blueprint created
```

### Enable manual backup

Before actually taking the backup we will have to enable the manual backup. Otherwise this is the problem that we see

```
Failed while waiting for Pod kanister-job-mbl6v to complete: Pod failed to complete in time: Pod failed or did not transition into complete state: Manual backups are not enabled. Make sure that \"spec.backups.pgbackrest.manual\" config is set PostgresCluster resource: Pod kanister-job-mbl6v failed
```

Before actually taking the backup, we will hav to enable the manual backup using the postgrescluster resource. Follow this to do that
https://docs.kasten.io/latest/kanister/pgo/logical.html#enable-manual-backups-on-postgrescluster


```
{"ActionSet":"backup-8kv4j","File":"pkg/controller/controller.go","Function":"github.com/kanisterio/kanister/pkg/controller.(*Controller).handleActionSet","Line":408,"cluster_name":"aa5ba13d-e0f9-40c5-a8a9-14d1a55a5302","error":"could not fetch object name: hippo, namespace: postgres-operator, group: postgres-operator.crunchydata.com/, version: v1beta1, resource: postgresclusters: postgresclusters.postgres-operator.crunchydata.com \"hippo\" is forbidden: User \"system:serviceaccount:kanister:kanister-kanister-operator\" cannot get resource \"postgresclusters\" in API group \"postgres-operator.crunchydata.com\" in the namespace \"postgres-operator\"","hostname":"kanister-kanister-operator-77999cf884-njfrt","level":"info","msg":"Failed to launch Action backup-8kv4j:","time":"2023-10-27T12:40:27.181426288Z"}
```

Edit cluster role

```
- apiGroups:
 41   - postgres-operator.crunchydata.com
 42   resources:
 43   - postgresclusters
 44   verbs:
 45   - get
```

after editing the cluster role, ran the actionset again and started getting this error

```
{"ActionSet":"backup-pbdb9","Container":"container","File":"pkg/output/stream.go","Function":"github.com/kanisterio/kanister/pkg/output.splitLines","Line":46,"LogKind":"datapath","Phase":"pgoBackup","Pod":"kanister-job-zmvch","Pod_Out":"Error from server (Forbidden): postgresclusters.postgres-operator.crunchydata.com \"hippo\" is forbidden: User \"system:serviceaccount:kanister:kanister-kanister-operator\" cannot patch resource \"postgresclusters\" in API group \"postgres-operator.crunchydata.com\" in the namespace \"postgres-operator\"","cluster_name":"aa5ba13d-e0f9-40c5-a8a9-14d1a55a5302","hostname":"kanister-kanister-operator-77999cf884-njfrt","level":"info","msg":"","time":"2023-10-27T13:04:52.34019081Z"}
```

edit cluster role again to give patch permissions



### insert some data

https://access.crunchydata.com/documentation/postgres-operator/latest/tutorials/basic-setup/connect-cluster#connect-using-a-port-forward

if it doesn't work set envs using export and separately.

```
export PG_CLUSTER_PRIMARY_POD=$(kubectl get pod -n postgres-operator -o name \
  -l postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/role=master)
kubectl -n postgres-operator port-forward "${PG_CLUSTER_PRIMARY_POD}" 5432:5432

export PG_CLUSTER_USER_SECRET_NAME=hippo-pguser-hippo

export PGPASSWORD=$(kubectl get secrets -n postgres-operator "${PG_CLUSTER_USER_SECRET_NAME}" -o go-template='{{.data.password | base64decode}}')
export PGUSER=$(kubectl get secrets -n postgres-operator "${PG_CLUSTER_USER_SECRET_NAME}" -o go-template='{{.data.user | base64decode}}')
export PGDATABASE=$(kubectl get secrets -n postgres-operator "${PG_CLUSTER_USER_SECRET_NAME}" -o go-template='{{.data.dbname | base64decode}}')
psql -h localhost
```


```
kanctl create actionset --action backup --namespace kanister --blueprint pgo-blueprint --objects postgres-operator.crunchydata.com/v1beta1/postgresclusters/postgres-operator/hippo
Warning: Neither --profile nor --repository-server flag is provided.
Action might fail if blueprint is using these resources.
actionset backup-rw9qk created

```

And the actionset got completed.

```bash
k get actionsets.cr.kanister.io -n kanister backup-rw9qk
NAME           PROGRESS   RUNNING PHASE   LAST TRANSITION TIME   STATE
backup-rw9qk   100                        2023-10-27T13:07:44Z   complete
```

Create the restore action

```bash
kanctl create actionset --action restore --from=backup-rw9qk -n kanister
Warning: Neither --profile nor --repository-server flag is provided.
Action might fail if blueprint is using these resources.
actionset restore-backup-rw9qk-9xdpt created
```

restore action fails while waiting for restore to succeed. We have two phases in restore action. One actually starts the restore and other waits for the restore to complete.
After first phase is run a pod is created in `postgres-operator` namespace to actually do the restore. And that pods fails with below error


```bash

k logs -n postgres-operator hippo-pgbackrest-restore-4nslg -f
Defaulted container "pgbackrest-restore" out of: pgbackrest-restore, nss-wrapper-init (init)
+ pgbackrest restore --type=time '--target=2023-11-30 06:48:49.669295+00' --target-timeline=1 --stanza=db --pg1-path=/pgdata/pg15 --repo=1 --delta --target-action=promote --link-map=pg_wal=/pgdata/pg15_wal
2023-11-30 06:51:46.377 GMT [18] LOG:  starting PostgreSQL 15.4 on x86_64-pc-linux-gnu, compiled by gcc (GCC) 8.5.0 20210514 (Red Hat 8.5.0-18), 64-bit
2023-11-30 06:51:46.379 GMT [18] LOG:  listening on IPv6 address "::1", port 5432
2023-11-30 06:51:46.379 GMT [18] LOG:  listening on IPv4 address "127.0.0.1", port 5432
2023-11-30 06:51:46.388 GMT [18] LOG:  listening on Unix socket "/tmp/.s.PGSQL.5432"
2023-11-30 06:51:46.400 GMT [21] LOG:  database system was interrupted; last known up at 2023-11-30 06:48:38 GMT
2023-11-30 06:51:46.563 GMT [21] LOG:  starting point-in-time recovery to 2023-11-30 06:48:49.669295+00
2023-11-30 06:51:46.690 GMT [21] LOG:  restored log file "000000010000000000000006" from archive
2023-11-30 06:51:46.745 GMT [21] LOG:  redo starts at 0/6000028
2023-11-30 06:51:46.864 GMT [21] LOG:  restored log file "000000010000000000000007" from archive
2023-11-30 06:51:47.030 GMT [21] LOG:  restored log file "000000010000000000000008" from archive
2023-11-30 06:51:47.111 GMT [21] LOG:  consistent recovery state reached at 0/7000088
2023-11-30 06:51:47.111 GMT [18] LOG:  database system is ready to accept read-only connections
2023-11-30 06:51:47.155 GMT [21] LOG:  redo done at 0/8000060 system usage: CPU: user: 0.00 s, system: 0.01 s, elapsed: 0.40 s
2023-11-30 06:51:47.155 GMT [21] FATAL:  recovery ended before configured recovery target was reached
2023-11-30 06:51:47.158 GMT [18] LOG:  startup process (PID 21) exited with exit code 1
2023-11-30 06:51:47.158 GMT [18] LOG:  terminating any other active server processes
2023-11-30 06:51:47.160 GMT [18] LOG:  shutting down due to startup process failure
2023-11-30 06:51:47.164 GMT [18] LOG:  database system is shut down
psql: error: connection to server on socket "/tmp/.s.PGSQL.5432" failed: No such file or directory
	Is the server running locally and accepting connections on that socket?
2023-11-30 06:51:48.271 GMT [39] LOG:  starting PostgreSQL 15.4 on x86_64-pc-linux-gnu, compiled by gcc (GCC) 8.5.0 20210514 (Red Hat 8.5.0-18), 64-bit
2023-11-30 06:51:48.272 GMT [39] LOG:  listening on IPv6 address "::1", port 5432
2023-11-30 06:51:48.272 GMT [39] LOG:  listening on IPv4 address "127.0.0.1", port 5432
2023-11-30 06:51:48.281 GMT [39] LOG:  listening on Unix socket "/tmp/.s.PGSQL.5432"
2023-11-30 06:51:48.293 GMT [42] LOG:  database system was interrupted while in recovery at log time 2023-11-30 06:48:38 GMT
2023-11-30 06:51:48.293 GMT [42] HINT:  If this has occurred more than once some data might be corrupted and you might need to choose an earlier recovery target.
2023-11-30 06:51:48.416 GMT [42] LOG:  starting point-in-time recovery to 2023-11-30 06:48:49.669295+00
2023-11-30 06:51:48.548 GMT [42] LOG:  restored log file "000000010000000000000006" from archive
2023-11-30 06:51:48.598 GMT [42] LOG:  redo starts at 0/6000028
2023-11-30 06:51:48.734 GMT [42] LOG:  restored log file "000000010000000000000007" from archive
2023-11-30 06:51:48.899 GMT [42] LOG:  restored log file "000000010000000000000008" from archive
2023-11-30 06:51:48.978 GMT [42] LOG:  consistent recovery state reached at 0/7000088
2023-11-30 06:51:48.978 GMT [39] LOG:  database system is ready to accept read-only connections
2023-11-30 06:51:49.013 GMT [42] LOG:  redo done at 0/8000060 system usage: CPU: user: 0.00 s, system: 0.01 s, elapsed: 0.41 s
2023-11-30 06:51:49.013 GMT [42] FATAL:  recovery ended before configured recovery target was reached
2023-11-30 06:51:49.016 GMT [39] LOG:  startup process (PID 42) exited with exit code 1
2023-11-30 06:51:49.016 GMT [39] LOG:  terminating any other active server processes
2023-11-30 06:51:49.017 GMT [39] LOG:  shutting down due to startup process failure
2023-11-30 06:51:49.020 GMT [39] LOG:  database system is shut down
pg_ctl: could not start server
Examine the log output.

```


