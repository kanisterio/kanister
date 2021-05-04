# Kafka topic backup and restore
To Backup Kafka topic data, we have used Amazon S3 Sink connector which periodically polls data from Kafka and in turn uploads it to S3. Each chunk of data is represented as an S3 object. The key name encodes the topic, Kafka partition, and the start offset of this data chunk. If no partitioner is specified in the configuration, the default partitioner which preserves Kafka partitioning is used. The size of each data chunk is determined by the number of records written to S3 and schema compatibility.

To restore Kafka topic data, we have used Amazon S3 Source Connector that reads data exported to S3 by the Connect S3 Sink Connector and publishes it back to a Kafka topic. Depending on the format and partitioner used to write the data to S3, this connector can write to the destination topic using the same partitions as the original messages exported to S3 and maintain the same message order. Configuration is setup to mirror the Kafka Connect S3 Sink Connector and should be possible to make only minor changes to the original sink configuration.

Confluent Amazon S3 Source connector comes with a 30-day trial period without a license key. After 30 days, this connector is available under a Confluent enterprise license.

During restore, topic messages are purged before the restore operation is performed.

## Prerequisites

* Kubernetes 1.9+
* Kanister controller version 0.51.2 installed in the cluster in a namespace <kanister-operator-namespace>. This example uses `kasten-io` namespace
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## Assumption

* No consumer is consuming the topic at the moment topic is being restored.

## Setup Kafka
Kafka can be deployed via a helm chart https://bitnami.com/stack/kafka/helm, or via an operator like strimzi.io.
This example is deploying Kafka via Strimzi Operator.

```bash
# Create a namespace named kafka
$ kubectl create namespace kafka

# Deploying kafka via an operator strimzi.io.
$ kubectl apply -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka

# Provision the Apache kafka and zookeeper.
$ kubectl create -f ./kafka-cluster.yaml -n kafka

# Wait for the pods to be in ready state
$ kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka

# Setup kafdrop for monitoring the Kafka cluster
$ kubectl create -f kafdrop.yaml -n kafka

# By default, kafdrop runs on port 9000
kubectl port-forward kafdrop 7000:9000 -n kafka
```

## Validate producer and consumer
Create Producer and Consumer using Kafka image provided by strimzi.
```bash
# create a producer and push data to it
$ kubectl -n kafka run kafka-producer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-console-producer.sh --broker-list my-cluster-kafka-bootstrap:9092 --topic blogpost
> event1
> event2
> event3

# creating a consumer on a different terminal
$ kubectl -n kafka run kafka-consumer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-console-consumer.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --topic my-topic --from-beginning
```

**NOTE:**
* Here, we now have Kafka running with the broker running on service `my-cluster-kafka-bootstrap:9092`
* `s3Sink.properties` file contains properties related `Confluent s3 sink Connector`
* `s3Source.properties` file contains properties related `Confluent s3 source Connector`
* `kafkaConfiguration.properties` contains properties pointing to Kafka server

## Setup Blueprint, ConfigMap and S3 Location profile
Before setting up the Blueprint, a Kanister Profile is created with S3 details along with a ConfigMap with the configuration details.
```bash
# Create ConfigMap with the Properties file, s3 properties and kafkaConfiguration.properties
$ kubectl create configmap s3config --from-file=./s3Sink.properties --from-file=./kafkaConfiguration.properties --from-file=./s3Source.properties -n kafka

# Create Profile pointing to s3 bucket
$ kanctl create profile s3compliant --access-key <aws-access-key> \
        --secret-key <aws-secret-key> \
        --bucket <aws-bucket-name> --region <aws-region-name> \
        --namespace kafka

# Blueprint Definition
$ kubectl create -f ./kafka-blueprint.yaml -n kasten-io
```
## Perform Backup
To perform backup to S3, an ActionSet is created which to run `kafka-connect`.
```bash
# Create an actionset
$ kanctl create actionset --action backup --namespace kasten-io --blueprint kafka-blueprint --profile kafka/s3-profile-fn64h --objects v1/configmaps/kafka/s3config
```
## Verify the backup
We can verify the backup operation by adding some data to the topic configured earlier

* List all topics in Kafka server
```bash
$ kubectl -n kafka run kafka-producer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-topics.sh --bootstrap-server=my-cluster-kafka-bootstrap:9092 --list

## Perform Restore
To perform restore, a pre-hook restore operation is performed which will purge all events from the topics in the Kafka cluster whose backups were performed previously.
```bash
# Create a restoreprehook actionset
$ kanctl create actionset --action restoreprehook --namespace kasten-io --blueprint kafka-blueprint --profile kafka/s3-profile-fn64h --objects v1/configmaps/kafka/s3config

```
**NOTE:**
* Here, the topic must be present in the Kafka cluster
* Before running pre-hook operation, confirm that no other consumer is consuming data from that topic

Perform the restore operation

```bash
# Perform a restore actionset
$ kanctl create actionset --action restore --namespace kasten-io --blueprint kafka-blueprint --profile kafka/s3-profile-fn64h --objects v1/configmaps/kafka/s3config

```
## Verify restore
Create a consumer for topics
```bash
# Creating a consumer on a different terminal
$ kubectl -n kafka run kafka-consumer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-console-consumer.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --topic blogpost --from-beginning
```

### Troubleshooting

The following debug commands can be used to troubleshoot issues during the backup and restore processes:

Check Kanister controller logs:
```bash
$ kubectl --namespace kasten-io logs -l run=kanister-svc -f
```
Check events of the ActionSet:
```bash
$ kubectl describe actionset <actionset-name> -n kasten-io
```
