# MongoDB + Kanister sidecar Helm Chart

## Prerequisites Details
* Kubernetes 1.8+ with Beta APIs enabled.
* PV support on the underlying infrastructure.
* Kanister version 0.24.0 with `profiles.cr.kanister.io` CRD installed

## StatefulSet Details
* https://kubernetes.io/docs/concepts/abstractions/controllers/statefulsets/

## StatefulSet Caveats
* https://kubernetes.io/docs/concepts/abstractions/controllers/statefulsets/#limitations

## Chart Details

This chart implements a dynamically scalable [MongoDB replica set](https://docs.mongodb.com/manual/tutorial/deploy-replica-set/)
using Kubernetes StatefulSets and Init Containers.

## Installing the Chart

For basic installation, you can install using a the provided Helm chart that will install an instance of a
MongoDB ReplicaSet (a StatefulSet with a persistent volumes) as well as a Kanister blueprint to be used with it.

Prior to install you will need to have the Kanister Helm repository added to your local setup.

```bash
$ helm repo add kanister http://charts.kanister.io
```

Then install the sample MongoDB application with the release name `my-release` in its own namespace
`mongo-test` using the command below. Make sure you have the kanister controller running in namespace `kasten-io` which is the default setting in MongoDB charts. Otherwise, you will also have to set the `kanister.controller_namespace` parameter value to the respective kanister controller namespace in the following command:

```bash
# Replace the default s3 credentials (endpoint, bucket and region) with your credentials before you run this command
$ helm install kanister/kanister-mongodb-replicaset -n my-release --namespace mongo-test \
     --set profile.create='true' \
     --set profile.profileName='mongo-test-profile' \
     --set profile.location.type='s3Compliant' \
     --set profile.location.bucket='kanister-bucket' \
     --set profile.location.endpoint='https://my-custom-s3-provider:9000' \
     --set profile.location.region=us-west-2 \
     --set profile.aws.accessKey="${AWS_ACCESS_KEY_ID}" \
     --set profile.aws.secretKey="${AWS_SECRET_ACCESS_KEY}"
```

The command deploys MongoDB ReplicaSet on the Kubernetes cluster in the default
configuration. The [configuration](#configuration) section lists the parameters that can be
configured during installation. It also installs a `profiles.cr.kanister.io` CRD named `mongo-test-profile` in `mongo-test` namespace.

The command will also configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference whether one created as part of the application install or
not. Support for creating an ActionSet as part of install is simply for convenience.
This CR can be shared between Kanister-enabled application instances so one option is to
only create as part of the first instance.

If not creating a Profile CR, it is possible to use an even simpler command.

```bash
$ helm install kanister/kanister-mongodb-replicaset -n my-release --namespace mongo-test
```

Once MongoDB is running, you can populate it with some data. Let's add a collection called "restaurants" to a test database:

```bash
# Connect to MongoDB by running a shell inside MongoDB's pod
$ kubectl exec --namespace mongo-test -i -t my-release-kanister-mongodb-replicaset-0  -- bash -l

# From inside the shell, use the mongo CLI to insert some data into the test database
$ mongo test --quiet --eval "db.restaurants.insert({'name' : 'Roys', 'cuisine' : 'Hawaiian', 'id' : '8675309'})"
WriteResult({ "nInserted" : 1 })

# View the restaurants data in the test database
$ mongo test --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5a1dd0719dcbfd513fecf87c"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
```

## Protect the Application

You can now take a backup of the MongoDB data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
$ kanctl create actionset --action backup --namespace kasten-io --blueprint my-release-kanister-mongodb-replicaset-blueprint --statefulset mongo-test/my-release-kanister-mongodb-replicaset --profile mongo-test/mongo-test-profile
actionset backup-llfb8 created

$ kubectl --namespace kasten-io get actionsets.cr.kanister.io
NAME                 AGE
backup-llfb8         2h

# View the status of the actionset
$ kubectl --namespace kasten-io describe actionset backup-llfb8
```

### Disaster strikes!

Let's say someone with fat fingers accidentally deleted the restaurants collection using the following command:
```bash
# Drop the restaurants collection
$ mongo test --quiet --eval "db.restaurants.drop()"
true
```

If you try to access this data in the database, you should see that it is no longer there:
```bash
$ mongo test --quiet --eval "db.restaurants.find()"
# No entries should be found in the restaurants collection
```

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
$ kanctl --namespace kasten-io create actionset --action restore --from "backup-llfb8"
actionset restore-backup-llfb8-64gqm created

# View the status of the ActionSet
kubectl --namespace kasten-io describe actionset restore-backup-llfb8-64gqm
```

You should now see that the data has been successfully restored to MongoDB!

```bash
$ mongo test --quiet --eval "db.restaurants.find()"
{ "_id" : ObjectId("5a1dd0719dcbfd513fecf87c"), "name" : "Roys", "cuisine" : "Hawaiian", "id" : "8675309" }
```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kasten-io create actionset --action delete --from "backup-llfb8"
actionset "delete-backup-llfb8-k9ncm" created

# View the status of the ActionSet
$ kubectl --namespace kasten-io describe actionset delete-backup-llfb8-k9ncm
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kasten-io logs -l app=kanister-operator
```

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.
To completely remove the release include the `--purge` flag.

## Configuration

The following table lists the configurable MongoDB Kanister blueprint and Profile CR parameters and their
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
| `kanister.controller_namespace` | (Optional) Specify the namespace where the Kanister controller is running. | `kasten-io` |

The following tables lists the configurable parameters of the mongodb chart and their default values.

| Parameter                           | Description                                                               | Default                                             |
| ----------------------------------- | ------------------------------------------------------------------------- | --------------------------------------------------- |
| `replicas`                          | Number of replicas in the replica set                                     | `3`                                                 |
| `replicaSetName`                    | The name of the replica set                                               | `rs0`                                               |
| `podDisruptionBudget`               | Pod disruption budget                                                     | `{}`                                                |
| `port`                              | MongoDB port                                                              | `27017`                                             |
| `installImage.repository`           | Image name for the install container                                      | `k8s.gcr.io/mongodb-install`                        |
| `installImage.tag`                  | Image tag for the install container                                       | `0.5`                                               |
| `installImage.pullPolicy`           | Image pull policy for the init container that establishes the replica set | `IfNotPresent`                                      |
| `image.repository`                  | MongoDB image name                                                        | `mongo`                                             |
| `image.tag`                         | MongoDB image tag                                                         | `3.6`                                               |
| `image.pullPolicy`                  | MongoDB image pull policy                                                 | `IfNotPresent`                                      |
| `podAnnotations`                    | Annotations to be added to MongoDB pods                                   | `{}`                                                |
| `securityContext`                   | Security context for the pod                                              | `{runAsUser: 999, fsGroup: 999, runAsNonRoot: true}`|
| `resources`                         | Pod resource requests and limits                                          | `{}`                                                |
| `persistentVolume.enabled`          | If `true`, persistent volume claims are created                           | `true`                                              |
| `persistentVolume.storageClass`     | Persistent volume storage class                                           |                                                     |
| `persistentVolume.accessMode`       | Persistent volume access modes                                            | `[ReadWriteOnce]`                                   |
| `persistentVolume.size`             | Persistent volume size                                                    | `10Gi`                                              |
| `persistentVolume.annotations`      | Persistent volume annotations                                             | `{}`                                                |
| `tls.enabled`                       | Enable MongoDB TLS support including authentication                       | `false`                                             |
| `tls.cacert`                        | The CA certificate used for the members                                   | Our self signed CA certificate                      |
| `tls.cakey`                         | The CA key used for the members                                           | Our key for the self signed CA certificate          |
| `metrics.enabled`                   | Enable Prometheus compatible metrics for pods and replicasets             | `false`                                             |
| `metrics.image.repository`          | Image name for metrics exporter                                           | `ssalaues/mongodb-exporter`                         |
| `metrics.image.tag`                 | Image tag for metrics exporter                                            | `0.6.1`                                             |
| `metrics.image.pullPolicy`          | Image pull policy for metrics exporter                                    | `IfNotPresent`                                      |
| `metrics.port`                      | Port for metrics exporter                                                 | `9216`                                              |
| `metrics.path`                      | URL Path to expose metrics                                                | `/metrics`                                          |
| `metrics.socketTimeout`             | Time to wait for a non-responding socket                                  | `3s`                                                |
| `metrics.syncTimeout`               | Time an operation with this session will wait before returning an error   | `1m`                                                |
| `metrics.prometheusServiceDiscovery`| Adds annotations for Prometheus ServiceDiscovery                          | `true`                                              |
| `auth.enabled`                      | If `true`, keyfile access control is enabled                              | `false`                                             |
| `auth.key`                          | Key for internal authentication                                           |                                                     |
| `auth.existingKeySecret`            | If set, an existing secret with this name for the key is used             |                                                     |
| `auth.adminUser`                    | MongoDB admin user                                                        |                                                     |
| `auth.adminPassword`                | MongoDB admin password                                                    |                                                     |
| `auth.metricsUser`                  | MongoDB clusterMonitor user                                               |                                                     |
| `auth.metricsPassword`              | MongoDB clusterMonitor password                                           |                                                     |
| `auth.existingAdminSecret`          | If set, and existing secret with this name is used for the admin user     |                                                     |
| `serviceAnnotations`                | Annotations to be added to the service                                    | `{}`                                                |
| `configmap`                         | Content of the MongoDB config file                                        |                                                     |
| `nodeSelector`                      | Node labels for pod assignment                                            | `{}`                                                |
| `affinity`                          | Node/pod affinities                                                       | `{}`                                                |
| `tolerations`                       | List of node taints to tolerate                                           | `[]`                                                |
| `livenessProbe`                     | Liveness probe configuration                                              | See below                                           |
| `readinessProbe`                    | Readiness probe configuration                                             | See below                                           |
| `extraVars`                         | Set environment variables for the main container                          | `{}`                                                |
| `extraLabels`                       | Additional labels to add to resources                                     | `{}`                                                |

*MongoDB config file*

All options that depended on the chart configuration are supplied as command-line arguments to `mongod`. By default,
the chart creates an empty config file. Entries may be added via  the `configmap` configuration value.

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```console
$ helm install --name my-release -f values.yaml kanister/kanister-mongodb-replicaset
```

> **Tip**: You can use the default [values.yaml](values.yaml)

Once you have all 3 nodes in running, you can run the "test.sh" script in this directory, which will insert a key into
the primary and check the secondaries for output. This script requires that the `$RELEASE_NAME` environment variable
be set, in order to access the pods.

## Authentication

By default, this chart creates a MongoDB replica set without authentication. Authentication can be
enabled using the parameter `auth.enabled`. Once enabled, keyfile access control is set up and an
admin user with root privileges is created. User credentials and keyfile may be specified directly.
Alternatively, existing secrets may be provided. The secret for the admin user must contain the
keys `user` and `password`, that for the key file must contain `key.txt`.  The user is created with
full `root` permissions but is restricted to the `admin` database for security purposes. It can be
used to create additional users with more specific permissions.

## TLS support

To enable full TLS encryption set `tls.enabled` to `true`. It is recommended to create your own CA by executing:

```console
$ openssl genrsa -out ca.key 2048
$ openssl req -x509 -new -nodes -key ca.key -days 10000 -out ca.crt -subj "/CN=mydomain.com"
```

After that paste the base64 encoded (`cat ca.key | base64 -w0`) cert and key into the fields `tls.cacert` and
`tls.cakey`. Adapt the configmap for the replicaset as follows:

```yml
configmap:
  storage:
    dbPath: /data/db
  net:
    port: 27017
    ssl:
      mode: requireSSL
      CAFile: /ca/tls.crt
      PEMKeyFile: /work-dir/mongo.pem
  replication:
    replSetName: rs0
  security:
    authorization: enabled
    clusterAuthMode: x509
    keyFile: /keydir/key.txt
```

To access the cluster you need one of the certificates generated during cluster setup in `/work-dir/mongo.pem` of the
certain container or you generate your own one via:

```console
$ cat >openssl.cnf <<EOL
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names
[alt_names]
DNS.1 = $HOSTNAME1
DNS.1 = $HOSTNAME2
EOL
$ openssl genrsa -out mongo.key 2048
$ openssl req -new -key mongo.key -out mongo.csr -subj "/CN=$HOSTNAME" -config openssl.cnf
$ openssl x509 -req -in mongo.csr \
    -CA $MONGOCACRT -CAkey $MONGOCAKEY -CAcreateserial \
    -out mongo.crt -days 3650 -extensions v3_req -extfile openssl.cnf
$ rm mongo.csr
$ cat mongo.crt mongo.key > mongo.pem
$ rm mongo.key mongo.crt
```

Please ensure that you exchange the `$HOSTNAME` with your actual hostname and the `$HOSTNAME1`, `$HOSTNAME2`, etc. with
alternative hostnames you want to allow access to the MongoDB replicaset. You should now be able to authenticate to the
mongodb with your `mongo.pem` certificate:

```console
$ mongo --ssl --sslCAFile=ca.crt --sslPEMKeyFile=mongo.pem --eval "db.adminCommand('ping')"
```

## Promethus metrics
Enabling the metrics as follows will allow for each replicaset pod to export Prometheus compatible metrics
on server status, individual replicaset information, replication oplogs, and storage engine.

```yaml
metrics:
  enabled: true
  image:
    repository: ssalaues/mongodb-exporter
    tag: 0.6.1
    pullPolicy: IfNotPresent
  port: 9216
  path: "/metrics"
  socketTimeout: 3s
  syncTimeout: 1m
  prometheusServiceDiscovery: true
  resources: {}
```

More information on [MongoDB Exporter](https://github.com/percona/mongodb_exporter) metrics available.

## Readiness probe
The default values for the readiness probe are:

```yaml
readinessProbe:
  initialDelaySeconds: 5
  timeoutSeconds: 1
  failureThreshold: 3
  periodSeconds: 10
  successThreshold: 1
```

## Liveness probe
The default values for the liveness probe are:

```yaml
livenessProbe:
  initialDelaySeconds: 30
  timeoutSeconds: 5
  failureThreshold: 3
  periodSeconds: 10
  successThreshold: 1
```

## Deep dive

Because the pod names are dependent on the name chosen for it, the following examples use the
environment variable `RELEASENAME`. For example, if the helm release name is `messy-hydra`, one would need to set the
following before proceeding. The example scripts below assume 3 pods only.

```console
export RELEASE_NAME=messy-hydra
```

### Cluster Health

```console
$ for i in 0 1 2; do kubectl exec $RELEASE_NAME-kanister-mongodb-replicaset-$i -- sh -c 'mongo --eval="printjson(db.serverStatus())"'; done
```

### Failover

One can check the roles being played by each node by using the following:
```console
$ for i in 0 1 2; do kubectl exec $RELEASE_NAME-kanister-mongodb-replicaset-$i -- sh -c 'mongo --eval="printjson(rs.isMaster())"'; done

MongoDB shell version: 3.6.3
connecting to: mongodb://127.0.0.1:27017
MongoDB server version: 3.6.3
{
	"hosts" : [
		"messy-hydra-mongodb-0.messy-hydra-mongodb.default.svc.cluster.local:27017",
		"messy-hydra-mongodb-1.messy-hydra-mongodb.default.svc.cluster.local:27017",
		"messy-hydra-mongodb-2.messy-hydra-mongodb.default.svc.cluster.local:27017"
	],
	"setName" : "rs0",
	"setVersion" : 3,
	"ismaster" : true,
	"secondary" : false,
	"primary" : "messy-hydra-mongodb-0.messy-hydra-mongodb.default.svc.cluster.local:27017",
	"me" : "messy-hydra-mongodb-0.messy-hydra-mongodb.default.svc.cluster.local:27017",
	"electionId" : ObjectId("7fffffff0000000000000001"),
	"maxBsonObjectSize" : 16777216,
	"maxMessageSizeBytes" : 48000000,
	"maxWriteBatchSize" : 1000,
	"localTime" : ISODate("2016-09-13T01:10:12.680Z"),
	"maxWireVersion" : 4,
	"minWireVersion" : 0,
	"ok" : 1
}


```
This lets us see which member is primary.

Let us now test persistence and failover. First, we insert a key (in the below example, we assume pod 0 is the master):
```console
$ kubectl exec $RELEASE_NAME-kanister-mongodb-replicaset-0 -- mongo --eval="printjson(db.test.insert({key1: 'value1'}))"

MongoDB shell version: 3.6.3
connecting to: mongodb://127.0.0.1:27017
{ "nInserted" : 1 }
```

Watch existing members:
```console
$ kubectl run --attach bbox --image=mongo:3.6 --restart=Never --env="RELEASE_NAME=$RELEASE_NAME" -- sh -c 'while true; do for i in 0 1 2; do echo $RELEASE_NAME-kanister-mongodb-replicaset-$i $(mongo --host=$RELEASE_NAME-kanister-mongodb-replicaset-$i.$RELEASE_NAME-kanister-mongodb-replicaset --eval="printjson(rs.isMaster())" | grep primary); sleep 1; done; done';

Waiting for pod default/bbox2 to be running, status is Pending, pod ready: false
If you don't see a command prompt, try pressing enter.
messy-hydra-mongodb-2 "primary" : "messy-hydra-mongodb-0.messy-hydra-mongodb.default.svc.cluster.local:27017",
messy-hydra-mongodb-0 "primary" : "messy-hydra-mongodb-0.messy-hydra-mongodb.default.svc.cluster.local:27017",
messy-hydra-mongodb-1 "primary" : "messy-hydra-mongodb-0.messy-hydra-mongodb.default.svc.cluster.local:27017",
messy-hydra-mongodb-2 "primary" : "messy-hydra-mongodb-0.messy-hydra-mongodb.default.svc.cluster.local:27017",
messy-hydra-mongodb-0 "primary" : "messy-hydra-mongodb-0.messy-hydra-mongodb.default.svc.cluster.local:27017",

```

Kill the primary and watch as a new master getting elected.
```console
$ kubectl delete pod $RELEASE_NAME-kanister-mongodb-replicaset-0

pod "messy-hydra-mongodb-0" deleted
```

Delete all pods and let the statefulset controller bring it up.
```console
$ kubectl delete po -l "app=kanister-mongodb-replicaset,release=$RELEASE_NAME"
$ kubectl get po --watch-only
NAME                    READY     STATUS        RESTARTS   AGE
messy-hydra-mongodb-0   0/1       Pending   0         0s
messy-hydra-mongodb-0   0/1       Pending   0         0s
messy-hydra-mongodb-0   0/1       Pending   0         7s
messy-hydra-mongodb-0   0/1       Init:0/2   0         7s
messy-hydra-mongodb-0   0/1       Init:1/2   0         27s
messy-hydra-mongodb-0   0/1       Init:1/2   0         28s
messy-hydra-mongodb-0   0/1       PodInitializing   0         31s
messy-hydra-mongodb-0   0/1       Running   0         32s
messy-hydra-mongodb-0   1/1       Running   0         37s
messy-hydra-mongodb-1   0/1       Pending   0         0s
messy-hydra-mongodb-1   0/1       Pending   0         0s
messy-hydra-mongodb-1   0/1       Init:0/2   0         0s
messy-hydra-mongodb-1   0/1       Init:1/2   0         20s
messy-hydra-mongodb-1   0/1       Init:1/2   0         21s
messy-hydra-mongodb-1   0/1       PodInitializing   0         24s
messy-hydra-mongodb-1   0/1       Running   0         25s
messy-hydra-mongodb-1   1/1       Running   0         30s
messy-hydra-mongodb-2   0/1       Pending   0         0s
messy-hydra-mongodb-2   0/1       Pending   0         0s
messy-hydra-mongodb-2   0/1       Init:0/2   0         0s
messy-hydra-mongodb-2   0/1       Init:1/2   0         21s
messy-hydra-mongodb-2   0/1       Init:1/2   0         22s
messy-hydra-mongodb-2   0/1       PodInitializing   0         25s
messy-hydra-mongodb-2   0/1       Running   0         26s
messy-hydra-mongodb-2   1/1       Running   0         30s


...
messy-hydra-mongodb-0 "primary" : "messy-hydra-mongodb-0.messy-hydra-mongodb.default.svc.cluster.local:27017",
messy-hydra-mongodb-1 "primary" : "messy-hydra-mongodb-0.messy-hydra-mongodb.default.svc.cluster.local:27017",
messy-hydra-mongodb-2 "primary" : "messy-hydra-mongodb-0.messy-hydra-mongodb.default.svc.cluster.local:27017",
```

Check the previously inserted key:
```console
$ kubectl exec $RELEASE_NAME-kanister-mongodb-replicaset-1 -- mongo --eval="rs.slaveOk(); db.test.find({key1:{\$exists:true}}).forEach(printjson)"

MongoDB shell version: 3.6.3
connecting to: mongodb://127.0.0.1:27017
{ "_id" : ObjectId("57b180b1a7311d08f2bfb617"), "key1" : "value1" }
```

### Scaling

Scaling should be managed by `helm upgrade`, which is the recommended way.
