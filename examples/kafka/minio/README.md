# Kafka topic backup and restore
In this example, we backup and restore Kafka topic data to MinIO, a high performance s3 compatible object store. We have used Adobe S3 Kafka connector which periodically polls data from Kafka and upload to minio. Each chunk of data is represented as an object. If no partitioner is specified in the configuration, the default partitioner which preserves Kafka partitioning is used.

**NOTE:**
During restore, topic messages are purged first before the restore operation is performed.

## Prerequisites

* Kubernetes 1.25+
* Kanister controller version 0.94.0 installed in the cluster in a namespace <kanister-operator-namespace>. This example uses `kanister` namespace
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## Assumption

* No Producer is creating records in the topic when a backup is performed.
* No consumer is consuming the topic when the topic is being restored.

## Installing the Chart

Install the Kafka Operator using the helm chart with release name `kafka-release` using the following commands:

```bash
# Add strimzi in your local chart repository
$ helm repo add strimzi https://strimzi.io/charts/

# Update your local chart repository
$ helm repo update

# Install the Kafka Operator (Helm Version 3)
$ kubectl create namespace kafka-test
$ helm install kafka-release strimzi/strimzi-kafka-operator --namespace kafka

```
## Setup Kafka

```bash
# Provision the Apache Kafka and zookeeper.
$ kubectl create -f ./my-kafka-cluster.yaml -n kafka

# wait for the pods to be in ready state
$ kubectl wait kafka/my-kafka-cluster --for=condition=Ready --timeout=300s -n kafka

## Validate producer and consumer

Create Producer and Consumer using Kafka image provided by strimzi.

```bash
# create a producer and push data to it
$ kubectl -n kafka run kafka-producer-news -ti \
        --image=quay.io/strimzi/kafka:0.23.0-kafka-2.8.0 \
        --rm=true --restart=Never -- bin/kafka-console-producer.sh \
        --broker-list my-kafka-cluster-kafka-bootstrap:9092 \
        --topic news

> {"channel":"CNN","title":"show-name-1","anchor":["anchor-name-1","anchor-name-2"]}
> {"channel":"MSNBC","title":"show-name-2","anchor":["anchor-name-3"]}
> {"channel":"FOX","title":"show-name-3","anchor":["anchor-name-5","anchor-name-6"]}
> {"channel":"CNBC","title":"show-name-4","anchor":["anchor-name-8","anchor-name-9"]}

# creating a consumer on a different terminal to view the messages pushes by producer
$ kubectl -n kafka run kafka-consumer-news -ti \
        --image=quay.io/strimzi/kafka:0.23.0-kafka-2.8.0 \
        --rm=true --restart=Never -- bin/kafka-console-consumer.sh \
        --bootstrap-server my-kafka-cluster-kafka-bootstrap:9092 \
        --topic news  --from-beginning
```

**NOTE:**
* We now have Kafka deployed with the broker running on service `my-kafka-cluster-kafka-bootstrap:9092`
  
* `minio-s3-sink.properties` file contains properties related `s3 sink Connector` for writing to MinIO Object store
* `minio-s3-source.properties` file contains properties related `s3 source Connector` for reading from MinIO Object store
* `kafka-configuration.properties` contains properties pointing to the Kafka server

## Configuration

The following configuration applies to source and sink connector.

| Config Name | Notes |
| ---------- | ----- |
| s3.endpoint | The MinIO instance endpoint URL  |
| s3.bucket | The name of the bucket to write. This key will be dynamically retrieved from profile |
| s3.region | The region in which s3 bucket is present. This can default to any valid AWS S3 region. Ex: us-west-1  |
| s3.prefix | Prefix added to all object keys stored in bucket to "namespace" them |
| s3.path_style | Force path-style access to bucket |
| topics | Comma separated list of kafka topics that need to be processed |
| task.max | Max number of tasks that should be run inside the connector |
| format | S3 File Format |
| compressed_block_size | Size of _uncompressed_ data to write to the file before rolling to a new block/chunk |

These additional configs apply to the kafka-connect:

| Config Key | Notes |
| ---------- | ----- |
| bootstrap.servers | Kafka broker address in the cluster |
| plugin.path | Connector jar location |

## Setup Blueprint, ConfigMap and S3 Location profile

Before setting up the Blueprint, a Kanister s3compliant Profile is created with MinIO details along with a ConfigMap with the configuration details. `timeinSeconds` denotes the time after which sink connector needs to stop running.

```bash
# Create a ConfigMap with the sink, source and Kafka Configuration properties file
$ kubectl create configmap s3config \
        --from-file=minio-s3-sink.properties=./minio-s3-sink.properties \
        --from-file=minio-s3-source.properties=./minio-s3-source.properties \
        --from-file=kafka-configuration.properties=./kafka-configuration.properties \
        --from-literal=timeinSeconds=1800 -n kafka

# Create a s3compliant Profile pointing to MinIO bucket
$ kanctl create profile s3compliant --access-key <minio-access-key> \
        --secret-key <minio-secret-key> \
        --bucket <minio-bucket-name> \
        --region <any-valid-aws-region-name> \
        --namespace kafka

# Create the Blueprint
$ kubectl create -f ./kafka-blueprint-minio.yaml -n kanister
```

## Test the Blueprint

It's time to test the blueprint. 

* Create a topic `news` on the Kafka server. The `news` topic is configured as source and sink topic in `s3config` configmap.

```bash
$ kubectl -n kafka run kafka-producer -ti \
        --image=quay.io/strimzi/kafka:0.23.0-kafka-2.8.0 \
        --rm=true --restart=Never -- bin/kafka-topics.sh \
        --create --topic news --bootstrap-server my-kafka-cluster-kafka-bootstrap:9092
```

* Create a producer and push data to `news` topic

```bash
$ kubectl -n kafka run kafka-producer -ti --image=quay.io/strimzi/kafka:0.23.0-kafka-2.8.0 --rm=true --restart=Never -- bin/kafka-console-producer.sh --broker-list my-kafka-cluster-kafka-bootstrap:9092 --topic news

> {"channel":"CNN","title":"show-name-1","anchor":["anchor-name-1","anchor-name-2"]}
> {"channel":"MSNBC","title":"show-name-2","anchor":["anchor-name-3"]}
> {"channel":"FOX","title":"show-name-3","anchor":["anchor-name-5","anchor-name-6"]}
> {"channel":"CNBC","title":"show-name-4","anchor":["anchor-name-8","anchor-name-9"]}
```

## Perform Backup

To perform a backup to Minio bucket, an ActionSet is created which runs `adobe kafka connect image`.

```bash
# Get the name of the s3complaint profile created before
$ kubectl get profile.cr.kanister.io -n kafka

# Create an actionset and refer the profile and s3config configmap
$ kanctl create actionset --action backup --namespace kanister \
        --blueprint kafka-blueprint-minio --profile kafka/s3-profile-jlm9c \
        --objects v1/configmaps/kafka/s3config
```

### Disaster strikes!

Let's say someone accidentally removed the events from the `news` topic in the Kafka cluster:

```bash
# No events from `news` topic.
$ kubectl -n kafka run kafka-consumer -ti --image=quay.io/strimzi/kafka:0.23.0-kafka-2.8.0 --rm=true --restart=Never -- bin/kafka-console-consumer.sh --bootstrap-server my-kafka-cluster-kafka-bootstrap:9092 --topic news --from-beginning

```
## Perform Restore

To perform restore, a pre-hook restore operation is performed which will purge all events from the topics in the Kafka cluster whose backups were performed previously.

**NOTE:**
* Here, the topic must be present in the Kafka cluster
* Before running pre-hook operation, confirm that no other consumer is consuming data from that topic

```bash
# Get the name of the backup action to refer the backup that will be used to restore.
$ kanctl get actionset -n kanister

# Create an actionset and refer the backup, profile and s3config configmap
$ kanctl create actionset --action restore --from "backup-nqhmw" \
        --namespace kanister --blueprint kafka-blueprint-minio \
        --profile kafka/s3-profile-jlm9c \
        --objects v1/configmaps/kafka/s3config
```

## Verify restore

Create a consumer for topics

```bash
# Creating a consumer on a different terminal
$ kubectl -n kafka run kafka-consumer -ti \
        --image=quay.io/strimzi/kafka:0.23.0-kafka-2.8.0 \
        --rm=true --restart=Never -- bin/kafka-console-consumer.sh \
        --bootstrap-server my-kafka-cluster-kafka-bootstrap:9092 \
        --topic news --from-beginning

{"channel":"CNN","title":"show-name-1","anchor":["anchor-name-1","anchor-name-2"]}
{"channel":"MSNBC","title":"show-name-2","anchor":["anchor-name-3"]}
{"channel":"FOX","title":"show-name-3","anchor":["anchor-name-5","anchor-name-6"]}
{"channel":"CNBC","title":"show-name-4","anchor":["anchor-name-8","anchor-name-9"]}

```
All the messages restored can be viewed.

## Delete the Artifacts

The artifacts created by the backup action can be cleaned using the following command:

```bash
$ kanctl create actionset --action delete \
      --from "backup-hp4gb" \
      --namespace kanister --blueprint kafka-blueprint-minio \
      --profile kafka/s3-profile-jlm9c \
      --objects v1/configmaps/kafka/s3config
```

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io delete-backup-hp4gb-khq4t
NAME                        PROGRESS   RUNNING PHASE   LAST TRANSITION TIME   STATE
delete-backup-hp4gb-khq4t   10.00                      2023-08-21T18:58:45Z   complete
```

## Delete Blueprint and Profile CR

```bash
# Delete the blueprint
$ kubectl delete blueprints.cr.kanister.io kafka-blueprint-minio -n kanister

# Get the profile
$ kubectl get profiles.cr.kanister.io -n kafka
NAME               AGE
s3-profile-fn64h   2h

# Delete the profile
$ kubectl delete profiles.cr.kanister.io s3-profile-jlm9c -n kafka

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
$ kubectl get pod -n kafka

# Check the logs
$ kubectl logs <name-of-pod-running the job> -n kafka

```
