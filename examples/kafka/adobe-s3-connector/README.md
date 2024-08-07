# Kafka topic backup and restore
To backup and restore Kafka topic data, we have used Adobe S3 Kafka connector which periodically polls data from Kafka and in turn uploads it to S3. Each chunk of data is represented as an S3 object. If no partitioner is specified in the configuration, the default partitioner which preserves Kafka partitioning is used.

During restore, topic messages are purged before the restore operation is performed.

## Prerequisites

* Kubernetes 1.9+
* Kanister controller version 0.110.0 installed in the cluster in a namespace <kanister-operator-namespace>. This example uses `kanister` namespace
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## Assumption

* No consumer is consuming the topic at the moment topic is being restored.

## Installing the Chart

Install the Kafka Operator using the helm chart with release name `kafka-release` using the following commands:

```bash
# Add strimzi in your local chart repository
$ helm repo add strimzi https://strimzi.io/charts/

# Update your local chart repository
$ helm repo update

# Install the Kafka Operator (Helm Version 3)
$ kubectl create namespace kafka-test
$ helm install kafka-release strimzi/strimzi-kafka-operator --namespace kafka-test

```
## Setup Kafka

```bash
# Provision the Apache Kafka and zookeeper.
$ kubectl create -f ./kafka-cluster.yaml -n kafka-test

# wait for the pods to be in ready state
$ kubectl wait kafka-test/my-cluster --for=condition=Ready --timeout=300s -n kafka-test

# setup kafdrop for monitoring the Kafka cluster, this is not mandatory for the blueprint as a part of restore and backup.
$ kubectl create -f kafdrop.yaml -n kafka-test

# by default kafdrop run on port 9000, we can view it by
kubectl port-forward kafdrop 7000.91.0 -n kafka-test
```

## Validate producer and consumer

Create Producer and Consumer using Kafka image provided by strimzi.

```bash
# create a producer and push data to it
$ kubectl -n kafka-test run kafka-producer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-console-producer.sh --broker-list my-cluster-kafka-bootstrap:9092 --topic blogpost
> event1
> event2
> event3

# creating a consumer on a different terminal
$ kubectl -n kafka-test run kafka-consumer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-console-consumer.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --topic blogpost --from-beginning
```

**NOTE:**
* Here, we now have Kafka running with the broker running on service `my-cluster-kafka-bootstrap:9092`
* `adobe-s3-sink.properties` file contains properties related `s3 sink Connector`
* `adobe-s3-source.properties` file contains properties related `s3 source Connector`
* `kafkaConfiguration.properties` contains properties pointing to Kafka server

## Configuration

The following configuration applies to source and sink connector.

| Config Key | Notes |
| ---------- | ----- |
| name | name of the connector |
| s3.bucket | The name of the bucket to write. This key will be dynamically added from profile |
| s3.region | The region in which s3 bcuket is present. This key will be dynamically added from profile |
| s3.prefix | Prefix added to all object keys stored in bucket to "namespace" them |
| s3.path_style | Force path-style access to bucket |
| topics | Comma separated list of topics that need to be processed |
| task.max | Max number of tasks that should be run inside the connector |
| format | S3 File Format |
| compressed_block_size | Size of _uncompressed_ data to write to the file before rolling to a new block/chunk |

These additional configs apply to the kafka-connect:

| Config Key | Notes |
| ---------- | ----- |
| bootstrap.servers | Kafka broker address in the cluster |
| plugin.path | Connector jar location |

## Setup Blueprint, ConfigMap and S3 Location profile

Before setting up the Blueprint, a Kanister Profile is created with S3 details along with a ConfigMap with the configuration details. `timeinSeconds` denotes the time after which sink connector needs to stop running.

```bash
# Create ConfigMap with the properties file, S3 properties and kafkaConfiguration.properties
$ kubectl create configmap s3config --from-file=adobe-s3-sink.properties=./adobe-s3-sink.properties --from-file=adobe-kafkaConfiguration.properties=./adobe-kafkaConfiguration.properties --from-file=adobe-s3-source.properties=./adobe-s3-source.properties --from-literal=timeinSeconds=1800 -n kafka-test

# Create Profile pointing to S3 bucket
$ kanctl create profile s3compliant --access-key <aws-access-key> \
        --secret-key <aws-secret-key> \
        --bucket <aws-bucket-name> --region <aws-region-name> \
        --namespace kafka-test

# Blueprint Definition
$ kubectl create -f ./kafka-blueprint.yaml -n kanister
```

## Insert Data in Topic

* Create a topic `blogs` on the Kafka server. The `blogs` topic is configured as source and sink topic in `s3config` configmap

```bash
$ kubectl -n kafka-test run kafka-producer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-topics.sh --create --topic blogpost --bootstrap-server my-cluster-kafka-bootstrap:9092
```

* Create a producer to push data to `blogs` topic

```bash
$ kubectl -n kafka-test run kafka-producer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-console-producer.sh --broker-list my-cluster-kafka-bootstrap:9092 --topic blogs

>{"title":"The Matrix","year":1999,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
>{"title":"ABCD3","year":2000,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
>{"title":"Student of the year","year":2001,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
>{"title":"ABCD","year":2002,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
```

## Perform Backup

To perform backup to S3, an ActionSet is created which runs `kafka-connect`.

```bash
# Create an actionset
$ kanctl create actionset --action backup --namespace kanister --blueprint kafka-blueprint --profile kafka-test/s3-profile-fn64h --objects v1/configmaps/kafka-test/s3config
```

### Disaster strikes!

Let's say someone accidentally removed the events from the `blogs` topic in the Kafka cluster:

```bash
# No events from `blogs` topic.
$ kubectl -n kafka-test run kafka-consumer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-console-consumer.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --topic blogs --from-beginning

```
## Perform Restore

To perform restore, a pre-hook restore operation is performed which will purge all events from the topics in the Kafka cluster whose backups were performed previously.

```bash
$ kanctl create actionset --action restore --from "backup-rslmb" --namespace kanister --blueprint kafka-blueprint --profile kafka-test/s3-profile-fn64h --objects v1/configmaps/kafka-test/s3config

```
**NOTE:**
* Here, the topic must be present in the Kafka cluster
* Before running pre-hook operation, confirm that no other consumer is consuming data from that topic

## Verify restore

Create a consumer for topics

```bash
# Creating a consumer on a different terminal
$ kubectl -n kafka-test run kafka-consumer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-console-consumer.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --topic blogs --from-beginning

>{"title":"The Matrix","year":1999,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
>{"title":"ABCD3","year":2000,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
>{"title":"Student of the year","year":2001,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
>{"title":"ABCD","year":2002,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
```
All the messages restored can be viewed.

## Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from backup-rslmb --namespacetargets kanister
actionset delete-backup-rslmb-cq6bw created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io delete-backup-rslmb-cq6bw
NAME                        PROGRESS   LAST TRANSITION TIME   STATE
delete-backup-rslmb-cq6bw   100.00     2022-12-15T10:05:38Z   complete
```
## Delete Blueprint and Profile CR

```bash
# Delete the blueprint
$ kubectl delete blueprints.cr.kanister.io <blueprint-name> -n kanister
# Get the profile
$ kubectl get profiles.cr.kanister.io -n kafka-test
NAME               AGE
s3-profile-fn64h   2h
# Delete the profile
$ kubectl delete profiles.cr.kanister.io s3-profile-fn64h -n kafka-test
```

## Troubleshooting

The following debug commands can be used to troubleshoot issues during the backup and restore processes:

Check Kanister controller logs:

```bash
$ kubectl --namespace kanister logs -l run=kanister-svc -f
```
Check events of the ActionSet:

```bash
$ kubectl describe actionset <actionset-name> -n kanister
```
Check the logs of the Kanister job

```bash
# Get the Kanister job pod name
$ kubectl get pod -n kafka-test

# Check the logs
$ kubectl logs <name-of-pod-running the job> -n kafka-test
```
