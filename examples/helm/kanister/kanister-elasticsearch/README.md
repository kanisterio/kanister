# Elasticsearch Helm Chart

This chart uses a standard Docker image of Elasticsearch (docker.elastic.co/elasticsearch/elasticsearch-oss) verison 6.3.1 and uses a service pointing to the master's transport port for service discovery.
Elasticsearch does not communicate with the Kubernetes API, hence no need for RBAC permissions.

## Warning for previous users
If you are currently using an earlier version of this Chart you will need to redeploy your Elasticsearch clusters. The discovery method used here is incompatible with using RBAC.
If you are upgrading to Elasticsearch 6 from the 5.5 version used in this chart before, please note that your cluster needs to do a full cluster restart.
The simplest way to do that is to delete the installation (keep the PVs) and install this chart again with the new version.
If you want to avoid doing that upgrade to Elasticsearch 5.6 first before moving on to Elasticsearch 6.0.

## Prerequisites Details

* Kubernetes 1.9+ with Beta APIs enabled.
* PV support on the underlying infrastructure.
* Kanister version 0.25.0 with `profiles.cr.kanister.io` CRD installed

## StatefulSets Details
* https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/

## StatefulSets Caveats
* https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#limitations

## Chart Details
This chart will do the following:

* Implement a dynamically scalable elasticsearch cluster using Kubernetes StatefulSets/Deployments and also add the kanister blueprint to be used with it.
* Multi-role deployment: master, client (coordinating) and data nodes
* Statefulset Supports scaling down without degrading the cluster

## Installing the Chart

For basic installation, you can install using the provided Helm chart that will install an instance of Elasticsearch as well as a Kanister blueprint to be used with it.

Prior to install you will need to have the Kanister Helm repository added to your local setup.

```bash
$ helm repo add kanister http://charts.kanister.io
```

Then install the sample Elasticsearch application with the release name `my-release` in its own namespace
`es-test` using the command below. Make sure you have the kanister controller running in namespace `kasten-io` which is the default setting in Elasticsearch charts. Otherwise, you will also have to set the `kanister.controller_namespace` parameter value to the respective kanister controller namespace in the following command:

```bash
# Replace the default s3 credentials (endpoint, bucket and region) with your credentials before you run this command
$ helm install kanister/kanister-elasticsearch -n my-release --namespace es-test \
     --set profile.create='true' \
     --set profile.profileName='es-test-profile' \
     --set profile.location.type='s3Compliant' \
     --set profile.location.bucket='kanister-bucket' \
     --set profile.location.endpoint='https://my-custom-s3-provider:9000' \
     --set profile.location.region=us-west-2 \
     --set profile.aws.accessKey="${AWS_ACCESS_KEY_ID}" \
     --set profile.aws.secretKey="${AWS_SECRET_ACCESS_KEY}"
```

The command deploys Elasticsearch on the Kubernetes cluster in the default
configuration. The [configuration](#configuration) section lists the parameters that can be
configured during installation. It also installs a `profiles.cr.kanister.io` CRD named `es-test-profile` in `es-test` namespace.

The command will also configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference whether one created as part of the application install or
not. Support for creating an ActionSet as part of install is simply for convenience.
This CR can be shared between Kanister-enabled application instances so one option is to
only create as part of the first instance.

If not creating a Profile CR, it is possible to use an even simpler command.

```bash
$ helm install kanister/kanister-elasticsearch -n my-release --namespace es-test
```

Once Elasticsearch is running, you can populate it with some data. Follow the instructions that get displayed by running command `helm status my-release` to connect to the application.

```bash
# Create index called customer
$ curl -X PUT "localhost:9200/customer?pretty"

# Add a customer named John Smith
$ curl -X PUT "localhost:9200/customer/_doc/1?pretty" -H 'Content-Type: application/json' -d'
{
  "name": "John Smith"
}
'

# View the data
$ curl -X GET "localhost:9200/_cat/indices?v"
health status index    uuid                   pri rep docs.count docs.deleted store.size pri.store.size
green  open   customer xbwj34pTSZOdDI7xVR0qIA   5   1          1            0      8.9kb          4.4kb
```

## Protect the Application

You can now take a backup of the Elasticsearch data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller using `kanctl`, a command-line tool that helps create ActionSets as shown below:

```bash
$ kanctl create actionset --action backup --namespace kasten-io --blueprint my-release-kanister-elasticsearch-blueprint --statefulset es-test/my-release-kanister-elasticsearch-data --profile es-test/es-test-profile

$ kubectl --namespace kasten-io get actionsets.cr.kanister.io
NAME                AGE
backup-lphk7        2h

# View the status of the actionset
$ kubectl --namespace kasten-io describe actionset backup-lphk7
```

## Disaster strikes!

Let's say someone with fat fingers accidentally deleted the customer index using the following command:

```bash
$ curl -X DELETE "localhost:9200/customer?pretty"
{
  "acknowledged" : true
}
```

If you try to access this data in the database, you should see that it is no longer there:

```bash
$ curl -X GET "localhost:9200/_cat/indices?v"
health status index uuid pri rep docs.count docs.deleted store.size pri.store.size
```

## Restore the Application

To restore the missing data, we want to use the backup created earlier in the steps above. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
$ kanctl --namespace kasten-io create actionset --action restore --from "backup-lphk7"
actionset restore-backup-lphk7-hndm6 created

# View the status of the ActionSet
kubectl --namespace kasten-io describe actionset restore-backup-lphk7-hndm6
```

You should now see that the data has been successfully restored to Elasticsearch!

```bash
$ curl -X GET "localhost:9200/_cat/indices?v"
health status index    uuid                   pri rep docs.count docs.deleted store.size pri.store.size
green  open   customer xbwj34pTSZOdDI7xVR0qIA   5   1          1            0      8.9kb          4.4kb
```

## Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kasten-io create actionset --action delete --from "backup-lphk7"
actionset "delete-backup-lphk7-5n8nz" created

# View the status of the ActionSet
$ kubectl --namespace kasten-io describe actionset delete-backup-lphk7-5n8nz
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kasten-io logs -l app=kanister-operator
```

## Delete the Helm deployment as normal

```
$ helm delete my-release
```

Deletion of the StatefulSet doesn't cascade to deleting associated PVCs. To delete them:

```
$ kubectl delete pvc -l release=my-release,component=data
```

## Configuration

The following table lists the configurable Elasticsearch Kanister blueprint and Profile CR parameters and their
default values. The Profile CR parameters are passed to the profile sub-chart.

| Parameter | Description | Default |
| --- | --- | --- |
| `profile.create` | (Optional) Specify if a Profile CR should be created as part of install. | ``false`` |
| `profile.defaultProfile` | (Optional if not creating a default Profile) Set to ``true`` to create a profile with name `default-profile` | ``false`` |
| `profile.profileName` | (Required if not creating a default Profile) Name for the profile that is created | `nil` |
| `profile.aws.accessKey` | (Required if creating profile) API Key for an s3 compatible object store. | `nil`|
| `profile.aws.secretKey` | (Required if creating profile) Corresponding secret for `accessKey`. | `nil` |
| `profile.location.bucket` | (Required if creating profile) A bucket that will be used to store Kanister artifacts. <br><br>The bucket must already exist and the account with the above API key and secret needs to have sufficient permissions to list, get, put, delete. | `nil` |
| `profile.location.region` | (Optional if creating profile) Region to be used for the bucket. | `nil` |
| `profile.location.endpoint` | (Optional if creating profile) The URL for an s3 compatible object store provider. Can be omitted if provider is AWS. Required for any other provider. | `nil` |
| `profile.verifySSL` | (Optional if creating profile) Set to ``false`` to disable SSL verification on the s3 endpoint. | `true` |
| `kanister.controller_namespace` | (Optional) Specify the namespace where the Kanister controller is running. | kasten-io |

The following table lists the configurable parameters of the elasticsearch chart and their default values.

|              Parameter               |                             Description                             |               Default                |
| ------------------------------------ | ------------------------------------------------------------------- | ------------------------------------ |
| `appVersion`                         | Application Version (Elasticsearch)                                 | `6.3.1`                              |
| `image.repository`                   | Container image name                                                | `docker.elastic.co/elasticsearch/elasticsearch-oss` |
| `image.tag`                          | Container image tag                                                 | `6.3.1`                              |
| `image.pullPolicy`                   | Container pull policy                                               | `Always`                             |
| `cluster.name`                       | Cluster name                                                        | `elasticsearch`                      |
| `cluster.kubernetesDomain`           | Kubernetes cluster domain name                                      | `cluster.local`                      |
| `cluster.xpackEnable`                | Writes the X-Pack configuration options to the configuration file   | `false`                              |
| `cluster.config`                     | Additional cluster config appended                                  | `{}`                                 |
| `cluster.env`                        | Cluster environment variables                                       | `{}`                                 |
| `client.name`                        | Client component name                                               | `client`                             |
| `client.replicas`                    | Client node replicas (deployment)                                   | `2`                                  |
| `client.resources`                   | Client node resources requests & limits                             | `{} - cpu limit must be an integer`  |
| `client.priorityClassName`           | Client priorityClass                                                | `nil`                                |
| `client.heapSize`                    | Client node heap size                                               | `512m`                               |
| `client.podAnnotations`              | Client Deployment annotations                                       | `{}`                                 |
| `client.nodeSelector`                | Node labels for client pod assignment                               | `{}`                                 |
| `client.tolerations`                 | Client tolerations                                                  | `{}`                                 |
| `client.serviceAnnotations`          | Client Service annotations                                          | `{}`                                 |
| `client.serviceType`                 | Client service type                                                 | `ClusterIP`                          |
| `master.exposeHttp`                  | Expose http port 9200 on master Pods for monitoring, etc            | `false`                              |
| `master.name`                        | Master component name                                               | `master`                             |
| `master.replicas`                    | Master node replicas (deployment)                                   | `2`                                  |
| `master.resources`                   | Master node resources requests & limits                             | `{} - cpu limit must be an integer`  |
| `master.priorityClassName`           | Master priorityClass                                                | `nil`                                |
| `master.podAnnotations`              | Master Deployment annotations                                       | `{}`                                 |
| `master.nodeSelector`                | Node labels for master pod assignment                               | `{}`                                 |
| `master.tolerations`                 | Master tolerations                                                  | `{}`                                 |
| `master.heapSize`                    | Master node heap size                                               | `512m`                               |
| `master.name`                        | Master component name                                               | `master`                             |
| `master.persistence.enabled`         | Master persistent enabled/disabled                                  | `true`                               |
| `master.persistence.name`            | Master statefulset PVC template name                                | `data`                               |
| `master.persistence.size`            | Master persistent volume size                                       | `4Gi`                                |
| `master.persistence.storageClass`    | Master persistent volume Class                                      | `nil`                                |
| `master.persistence.accessMode`      | Master persistent Access Mode                                       | `ReadWriteOnce`                      |
| `data.exposeHttp`                    | Expose http port 9200 on data Pods for monitoring, etc              | `false`                              |
| `data.replicas`                      | Data node replicas (statefulset)                                    | `3`                                  |
| `data.resources`                     | Data node resources requests & limits                               | `{} - cpu limit must be an integer`  |
| `data.priorityClassName`             | Data priorityClass                                                  | `nil`                                |
| `data.heapSize`                      | Data node heap size                                                 | `1536m`                              |
| `data.persistence.enabled`           | Data persistent enabled/disabled                                    | `true`                               |
| `data.persistence.name`              | Data statefulset PVC template name                                  | `data`                               |
| `data.persistence.size`              | Data persistent volume size                                         | `30Gi`                               |
| `data.persistence.storageClass`      | Data persistent volume Class                                        | `nil`                                |
| `data.persistence.accessMode`        | Data persistent Access Mode                                         | `ReadWriteOnce`                      |
| `data.podAnnotations`                | Data StatefulSet annotations                                        | `{}`                                 |
| `data.nodeSelector`                  | Node labels for data pod assignment                                 | `{}`                                 |
| `data.tolerations`                   | Data tolerations                                                    | `{}`                                 |
| `data.terminationGracePeriodSeconds` | Data termination grace period (seconds)                             | `3600`                               |
| `data.antiAffinity`                  | Data anti-affinity policy                                           | `soft`                               |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

In terms of Memory resources you should make sure that you follow that equation:

- `${role}HeapSize < ${role}MemoryRequests < ${role}MemoryLimits`

The YAML value of cluster.config is appended to elasticsearch.yml file for additional customization ("script.inline: on" for example to allow inline scripting)

# Deep dive

## Application Version

This chart aims to support Elasticsearch v2 and v5 deployments by specifying the `values.yaml` parameter `appVersion`.

### Version Specific Features

* Memory Locking *(variable renamed)*
* Ingest Node *(v5)*
* X-Pack Plugin *(v5)*

Upgrade paths & more info: https://www.elastic.co/guide/en/elasticsearch/reference/current/setup-upgrade.html

## Mlocking

This is a limitation in kubernetes right now. There is no way to raise the
limits of lockable memory, so that these memory areas won't be swapped. This
would degrade performance heavily. The issue is tracked in
[kubernetes/#3595](https://github.com/kubernetes/kubernetes/issues/3595).

```
[WARN ][bootstrap] Unable to lock JVM Memory: error=12,reason=Cannot allocate memory
[WARN ][bootstrap] This can result in part of the JVM being swapped out.
[WARN ][bootstrap] Increase RLIMIT_MEMLOCK, soft limit: 65536, hard limit: 65536
```

## Minimum Master Nodes
> The minimum_master_nodes setting is extremely important to the stability of your cluster. This setting helps prevent split brains, the existence of two masters in a single cluster.

>When you have a split brain, your cluster is at danger of losing data. Because the master is considered the supreme ruler of the cluster, it decides when new indices can be created, how shards are moved, and so forth. If you have two masters, data integrity becomes perilous, since you have two nodes that think they are in charge.

>This setting tells Elasticsearch to not elect a master unless there are enough master-eligible nodes available. Only then will an election take place.

>This setting should always be configured to a quorum (majority) of your master-eligible nodes. A quorum is (number of master-eligible nodes / 2) + 1

More info: https://www.elastic.co/guide/en/elasticsearch/guide/1.x/_important_configuration_changes.html#_minimum_master_nodes

# Client and Coordinating Nodes

Elasticsearch v5 terminology has updated, and now refers to a `Client Node` as a `Coordinating Node`.

More info: https://www.elastic.co/guide/en/elasticsearch/reference/5.5/modules-node.html#coordinating-node

## Select right storage class for SSD volumes

### GCE + Kubernetes 1.5

Create StorageClass for SSD-PD

```
$ kubectl create -f - <<EOF
kind: StorageClass
apiVersion: extensions/v1beta1
metadata:
  name: ssd
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-ssd
EOF
```
Create cluster with Storage class `ssd` on Kubernetes 1.5+

```bash
$ helm install kanister/kanister-elasticsearch -n my-release --namespace es-test \
     --set data.storageClass=ssd \
     --set data.storage=100Gi \
     --set profile.create='true' \
     --set profile.profileName='es-test-profile' \
     --set profile.location.type='s3Compliant' \
     --set profile.location.bucket='kanister-bucket' \
     --set profile.location.endpoint='https://my-custom-s3-provider:9000' \
     --set profile.location.region=us-west-2 \
     --set profile.aws.accessKey="${AWS_ACCESS_KEY_ID}" \
     --set profile.aws.secretKey="${AWS_SECRET_ACCESS_KEY}"
```
