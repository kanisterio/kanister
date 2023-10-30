


Create backup actionset

```bash
kanctl create actionset --action backup --namespace kanister --blueprint pgo-blueprint --objects postgres-operator.crunchydata.com/v1beta1/postgresclusters/postgres-operator/hippo
```


```
{"ActionSet":"backup-8kv4j","File":"pkg/controller/controller.go","Function":"github.com/kanisterio/kanister/pkg/controller.(*Controller).handleActionSet","Line":408,"cluster_name":"aa5ba13d-e0f9-40c5-a8a9-14d1a55a5302","error":"could not fetch object name: hippo, namespace: postgres-operator, group: postgres-operator.crunchydata.com/, version: v1beta1, resource: postgresclusters: postgresclusters.postgres-operator.crunchydata.com \"hippo\" is forbidden: User \"system:serviceaccount:kanister:kanister-kanister-operator\" cannot get resource \"postgresclusters\" in API group \"postgres-operator.crunchydata.com\" in the namespace \"postgres-operator\"","hostname":"kanister-kanister-operator-77999cf884-njfrt","level":"info","msg":"Failed to launch Action backup-8kv4j:","time":"2023-10-27T12:40:27.181426288Z"}
```

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

```bash

```