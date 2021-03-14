# kafka topic backup and restore
To Backup kafka topic data, we have used Amazon S3 Sink connector which periodically polls data from Kafka and in turn uploads it to S3. Each chunk of data is represented as an S3 object. The key name encodes the topic, the Kafka partition, and the start offset of this data chunk. If no partitioner is specified in the configuration, the default partitioner which preserves Kafka partitioning is used. The size of each data chunk is determined by the number of records written to S3 and by schema compatibility.

To restore kafka topic data, we have used Amazon S3 Source Connector that reads data exported to S3 by the Connect S3 Sink Connector and publishes it back to an Kafka topic. Depending on the format and partitioner used to write the data to S3, this connector can write to the destination topic using the same partitions as the original messages exported to S3 and maintain the same message order. Configuration is setup to mirror the Kafka Connect S3 Sink Connector and should be possible to make only minor changes to the original sink configuration.

Confluent Amazon S3 Source connector comes with a 30-day trial period without a license key. After 30 days, this connector is available under a Confluent enterprise license.

Topics messages are first purged and then restore operation is performed

## Prerequisites

* Kubernetes 1.9+
* K10 installed in your cluster, let's say in namespace `<kanister-operator-namespace>` Can be installed (https://docs.kasten.io/latest/install/install.html). in our case we have used `kasten-io` namespace
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## Assumption

* No consumer is consuming the topic at the moment topic is being restored.

## Setup Kafka
Kafka can be deployed via a helm chart https://bitnami.com/stack/kafka/helm, or via an operator like strimzi.io.
Here deploying Kafka via Strimzi Operator

```bash
# create namespace KAFKA
$ kubectl create namespace kafka

# Deploying kafka via an operator strimzi.io.
$ kubectl apply -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka

# Provision the Apache kafka and zookeeper.
$ kubectl create -f ./kafka-cluster.yaml -n kafka

# wait for the pods to be in ready state
$ kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka

# setup kafdrop for monitoring the kafka cluster
$ kubectl create -f kafdrop.yaml -n kafka

# by default kafdrop run on port 9000, we can view it by
kubectl port-forward kafdrop 7000:9000 -n kafka
```

## Validate producer and consumer
Create Producer and Consumer using kafka image provided by strimzi
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
* Here we have now kafka running with the broker running on service `my-cluster-kafka-bootstrap:9092`
* `s3Sink.properties` file contain properties related `Confluent s3 sink Connector`
* `s3Source.properties` file contain properties related `Confluent s3 source Connector`
* `kafkaConfiguration.properties` contain properties pointing to kafka server

## Setup Blueprint, configMap and location profile
Before Setting up Blueprint, a profile is created which has s3 Details, alongwith that a configMap with the configuration details
```bash
# Create ConfigMap with the Properties file s3 properties and kafkaConfiguration.properties
$ kubectl create configmap s3config --from-file=./s3Sink.properties --from-file=./kafkaConfiguration.properties --from-file=./s3Source.properties -n kafka

# Create Profile pointing to s3 bucket
$ kanctl create profile s3compliant --access-key <aws-access-key> \
        --secret-key <aws-secret-key> \
        --bucket <aws-bucket-name> --region <aws-region-name> \
        --namespace kafka
secret 's3-secret-gkvgi4' created
profile 's3-profile-fn64h' created

# Blueprint Definition
$ kubectl create -f ./kafka-blueprint.yaml -n kasten-io
```
## Perform Backup
To perform backup to s3, an actionset is created which will run kafka-connect
```bash
# Create an actionset
$ kanctl create actionset --action backup --namespace kasten-io --blueprint kafka-blueprint --profile kafka/s3-profile-fn64h --objects v1/configmaps/kafka/s3config
```
## Verify the backup
We can verify the backup operation by adding some data to the topic configured earlier

* lIst all topics in kafka server
```bash
$ kubectl -n kafka run kafka-producer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-topics.sh --bootstrap-server=my-cluster-kafka-bootstrap:9092 --list
```
* create a topic to kafka server
```bash
$ kubectl -n kafka run kafka-producer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-topics.sh --create --topic blogpost --bootstrap-server my-cluster-kafka-bootstrap:9092
```
* create a producer to push data to blogpost topic
```bash
$ kubectl -n kafka run kafka-producer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-console-producer.sh --broker-list my-cluster-kafka-bootstrap:9092 --topic blogpost

>{"title":"The Matrix","year":1999,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
>{"title":"ABCD3","year":2000,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
>{"title":"Student of the year","year":2001,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
>{"title":"ABCD","year":2002,"cast":["Keanu Reeves","Laurence Fishburne","Carrie-Anne Moss","Hugo Weaving","Joe Pantoliano"],"genres":["Science Fiction"]}
```
* check S3 bucket for the topic

## Perform Restore
To perform restore, a prehook restore operation is performed which will purge all events from the topics in the kafka cluster whose backups were performed previously.
```bash
# Create a restoreprehook actionset
$ kanctl create actionset --action restoreprehook --namespace kasten-io --blueprint kafka-blueprint --profile kafka/s3-profile-fn64h --objects v1/configmaps/kafka/s3config

```
**NOTE:**
* Here the topic need to be already present in the kafka cluster.
* Before running prehook operation confirm that no other consumer is consuming data from that topic

Perform the restore operation

```bash
# Perform a restore actionset
$ kanctl create actionset --action restore --namespace kasten-io --blueprint kafka-blueprint --profile kafka/s3-profile-fn64h --objects v1/configmaps/kafka/s3config

```
## Verify restore
Create a consumer for topics
```bash
# creating a consumer on a different terminal
$ kubectl -n kafka run kafka-consumer -ti --image=strimzi/kafka:0.20.0-kafka-2.6.0 --rm=true --restart=Never -- bin/kafka-console-consumer.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --topic blogpost --from-beginning
```
All the messages restored can be viewed

## Delete Blueprint and Profile CR

```bash
# delete the blueprint
$ kubectl delete blueprints.cr.kanister.io <blueprint-name> -n kasten-io
# Get the profile
$ kubectl get profiles.cr.kanister.io -n kafka
NAME               AGE
s3-profile-fn64h   2h
# Delete the profile
$ kubectl delete profiles.cr.kanister.io s3-profile-fn64h -n kafka
```

### Troubleshooting

If you run into any issues with the above commands,

you can check the logs of the controller using:
```bash
$ kubectl --namespace kasten-io logs -l run=kanister-svc -f
```
you can check events of the actionset:
```bash
$ kubectl describe actionset <actionset-name> -n kasten-io
```
you can also check the logs of kanister job
```bash
# get the pod name
$ kubectl get pod -n kafka

# check the logs
$ kubectl logs <name-of-pod-running the job> -n kafka
```
