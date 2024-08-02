# K8ssandra

[K8ssandra](https://k8ssandra.io/) is a cloud-native distribution of Apache Cassandra (Cassandra) designed to run on Kubernetes. K8ssandra follows the K8s operator pattern to automate operational tasks. This includes metric, data anti-entropy services, and backup/restore tooling. More details can be found [here](https://docs.k8ssandra.io).

K8ssandra operator uses Medusa to backup and restore Cassandra data. Kanister can make use of Medusa operator APIs to perform backup and restore of Cassandra data.

## Prerequisites

* Kubernetes 1.17+
* PV support on the underlying infrastructure
* Kanister controller version 0.110.0 installed in your cluster
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)
* K8ssandra needs at least 4 cores and 8GB of RAM available to Docker and appropriate heap sizes for Cassandra and Stargate. If you don’t have those resources available, you can avoid deploying features such as monitoring, Reaper and Medusa, and also reduce the number of Cassandra nodes.

## Chart Details

We will be using [K8ssandra](https://github.com/k8ssandra/k8ssandra/tree/main/charts/k8ssandra) official Helm charts to deploy K8ssandra stack on [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Installing the K8ssandra

> Complete guide to provision K8s cluster locally with required resources and installing K8ssandra can be found [here](https://docs-v1.k8ssandra.io/install/local/).

Before we begin, make sure that there exists a storageclass with **VOLUMEBINDINGMODE of WaitForFirstConsumer.**


### Create S3 creds secret for Medusa Amazon S3 integration

K8ssandra Operator deploys Medusa to support backup and restore operations of Apache Cassandra® tables.

Medusa supports different types of object stores including GCS, S3, and S3-compatible stores like MinIO, CEPH Object Gateway, etc. The complete list can be found [here](https://docs-v1.k8ssandra.io/components/medusa/#supported-storage-objects).

For the scope of this example, we will just focus on S3 bucket. But you can choose any object store of your choice.

Create a secret using the following template to enable S3 integration with Medusa. Replace `my_access_key` and `my_secret_key` with actual keys before creating the secret.

```
apiVersion: v1
kind: Secret
metadata:
 name: medusa-bucket-key
type: Opaque
stringData:
 # Note that this currently has to be set to medusa_s3_credentials!
 medusa_s3_credentials: |-
   [default]
   aws_access_key_id = my_access_key
   aws_secret_access_key = my_secret_key
```

### Create K8ssandra cluster
Create a Helm configuration file e.g values.yaml with the following configuration so that K8ssandra installation works on the limited resources we have locally.

```
cassandra:
  version: "3.11.10"
  cassandraLibDirVolume:
    size: 5Gi
  allowMultipleNodesPerWorker: true
  heap:
    size: 200M
    newGenSize: 800M
  resources:
    requests:
      cpu: 900m
      memory: 1Gi
    limits:
      cpu: 900m
      memory: 1Gi
  datacenters:
  - name: dc1
    size: 1
    racks:
    - name: default
kube-prometheus-stack:
  enabled: false
reaper:
  enabled: false
reaper-operator:
  enabled: false
stargate:
  enabled: true
  replicas: 1
  heapMB: 256
  cpuReqMillicores: 200
  cpuLimMillicores: 1000
medusa:
  enabled: true
  storageSecret: medusa-bucket-key
  storage: s3
```

Pass the values.yaml file to `helm install` command to overwrite the default Helm chart configuration

```bash
# Add k8ssandra Helm repo
$ helm repo add k8ssandra https://helm.k8ssandra.io/stable
$ helm repo update

# Install K8ssandra operator
$ helm install k8ssandra k8ssandra/k8ssandra -n k8ssandra --create-namespace -f ./values.yaml \
    --set cassandra.cassandraLibDirVolume.storageClass=<storage_class> \
    --set medusa.bucketName=<aws_s3_bucket_name> \
    --set medusa.storage_properties.region=<aws_bucket_region>
```

Where,

- `storage_class` is the storageclass with `WaitForFirstConsumer` VolumeBindingMode.
- `aws_s3_bucket_name` is the S3 bucket that will be used for backup by Medusa.
- `aws_bucket_region` is the AWS region bucket exists.

Installing the Helm chart will create a few components like `cass-operator`, `medusa-operator` and `dc` pods. The actual Cassandra node name from the running pod listing is `k8ssandra-dc1-rack-a-sts-0` which we’ll use throughout the following example.


**NOTE:**

Kanister operator role needs to be updated in order to access custom resources.
Create required ClusterRole and ClusterRoleBinding with the following command.

```bash
$ kubectl auth reconcile -f kanister-k8ssandra-rbac.yaml
```

You may have to update the service account name in the specs if you have deployed Kanister with another name

## Integrating with Kanister

If you have deployed the K8ssandra operator with other name than `k8ssandra` and namespace other than `k8ssandra`, you need to modify the commands used below to use the correct name and namespace.

### Create Blueprint

Create Blueprint in the same namespace as the controller

```bash
$ kubectl create -f k8ssandra-blueprint.yaml -n kanister
blueprint.cr.kanister.io/k8ssandra-blueprint created
```

Once K8ssandra pods are running, you can populate it with some data.

```bash
# Connect to cassandra pod
$ kubectl exec -ti k8ssandra-dc1-rack-a-sts-0 -c cassandra -n k8ssandra -- bash

# once you are inside the pod use `cqlsh` to get into the Cassandra CLI and run the below commands to create the keyspace
$ cassandra@k8ssandra-dc1-rack-a-sts-0:/$ cqlsh
Connected to k8ssandra at 127.0.0.1:9042.
[cqlsh 5.0.1 | Cassandra 3.11.10 | CQL spec 3.4.4 | Native protocol v4]
Use HELP for help.
$ cqlsh> create keyspace restaurants with replication  = {'class':'SimpleStrategy', 'replication_factor': 3};

# once the keyspace is created let's create a table named guests and some data into that table
$ cqlsh> create table restaurants.guests (id UUID primary key, firstname text, lastname text, birthday timestamp);
$ cqlsh> insert into restaurants.guests (id, firstname, lastname, birthday)  values (5b6962dd-3f90-4c93-8f61-eabfa4a803e2, 'Vivek', 'Singh', '2015-02-18');
$ cqlsh> insert into restaurants.guests (id, firstname, lastname, birthday)  values (5b6962dd-3f90-4c93-8f61-eabfa4a803e4, 'Prasad', 'Hemsworth', '2015-02-18');
$ cqlsh> insert into restaurants.guests (id, firstname, lastname, birthday)  values (5b6962dd-3f90-4c93-8f61-eabfa4a803e3, 'Tom', 'Singh', '2015-02-18');

# verify data
$ cqlsh> select * from restaurants.guests;

 id                                   | birthday                        | firstname | lastname
--------------------------------------+---------------------------------+-----------+-----------
 5b6962dd-3f90-4c93-8f61-eabfa4a803e2 | 2015-02-18 00:00:00.000000+0000 |     Vivek |     Singh
 5b6962dd-3f90-4c93-8f61-eabfa4a803e3 | 2015-02-18 00:00:00.000000+0000 |       Tom |     Singh
 5b6962dd-3f90-4c93-8f61-eabfa4a803e4 | 2015-02-18 00:00:00.000000+0000 |    Prasad | Hemsworth

(3 rows)

```

## Protect the Application

You can now take a backup of the K8ssandra cluster data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
# Create actionset to run backup action from the blueprint
$ kanctl create actionset --action backup --blueprint k8ssandra-blueprint --objects cassandra.datastax.com/v1beta1/cassandradatacenters/k8ssandra/dc1 -n kanister
actionset backup-swxnq created

# you can check the status of the actionset either by describing the actionset resource or by checking the kanister operator's pod log
$ kubectl describe actionset backup-swxnq -n kanister

# Wait till the actionset `completes`
```

### Disaster strikes!

Let's say someone accidentally deleted the bucket using the following command in K8ssandra cluster pod.

```bash
# Connect to cassandra pod
$ kubectl exec -ti k8ssandra-dc1-rack-a-sts-0 -c cassandra -n k8ssandra -- bash

# once you are inside the pod use `cqlsh` to get into the Cassandra CLI and run below commands to create the keyspace
$ cassandra@k8ssandra-dc1-rack-a-sts-0:/$ cqlsh
Connected to k8ssandra at 127.0.0.1:9042.
[cqlsh 5.0.1 | Cassandra 3.11.10 | CQL spec 3.4.4 | Native protocol v4]
Use HELP for help.

# once you are inside the pod use `cqlsh` to get into the Cassandra CLI and run below commands to create the keyspace
# drop the guests table
$ cqlsh> drop table if exists restaurants.guests;
# drop restaurants keyspace
$ cqlsh> drop  keyspace  restaurants;
```

### Restore the Application

Now that we have removed the data from the Cassandra database's table, let's restore the data using the backup that we have already created in the earlier step. To do that we will again create an Actionset resource but for restore instead of backup. You can create the Actionset using the below command

```bash
$ kanctl create actionset --action restore --from backup-swxnq -n kanister
actionset restore-backup-swxnq-9hhmd created

# View the status of the ActionSet
$ kubectl describe actionset restore-backup-swxnq-9hhmd --namespace kanister
```

Once you have verified that the status of the Actionset <restore-actionset> is completed. You can check if the data is restored or not by EXECing into the Cassandra pod and selecting all the data from the table.

```bash
$ kubectl exec -ti k8ssandra-dc1-rack-a-sts-0 -c cassandra -n k8ssandra -- bash
# once you are inside the pod use `cqlsh` to get into the Cassandra CLI and run below commands to create the keyspace
$ cqlsh> select * from restaurants.guests;
```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command.

```bash
$ kanctl create actionset --action delete --from backup-swxnq -n kanister
actionset delete-backup-swxnq-l5zc4 created


# View the status of the ActionSet
$ kubectl --namespace kanister describe actionset delete-backup-swxnq-l5zc4
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using.

```bash
$ kubectl --namespace <kanister-operator-namespace> logs -l app=kanister-operator
```

You can also check events of the ActionSet with the following command.

```bash
$ kubectl describe actionset <actionset-name> -n <kanister-operator-namespace>
```

## Uninstalling the Chart

To uninstall/delete the K8ssandra application run the below command.

```bash
$ helm delete k8ssandra -n k8ssandra
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Delete Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io k8ssandra-blueprint -n kanister
```
