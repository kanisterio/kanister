# Elasticsearch Helm Chart

This chart uses a standard Docker image of
[Elasticsearch](docker.elastic.co/elasticsearch/elasticsearch-oss) version
8.5.1 and uses a service pointing to the master's transport port for service
discovery. Elasticsearch does not communicate with the Kubernetes API,
hence no need for RBAC permissions.

## Warning for previous users
If you are currently using an earlier version of this Chart you will need to
redeploy your Elasticsearch clusters. The discovery method used here is
incompatible with using RBAC. If you are upgrading to Elasticsearch 6 from the
5.5 version used in this chart before, please note that your cluster needs to
do a full cluster restart. The simplest way to do that is to delete the
installation (keep the PVs) and install this chart again with the new version.
If you want to avoid doing that upgrade to Elasticsearch 5.6 first before
moving on to Elasticsearch 6.0.

## Prerequisites Details

* Kubernetes 1.20+
* PV provisioner support in the underlying infrastructure
* Kanister controller version 0.110.0 installed in your cluster
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)

## StatefulSets Details
* https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/

## StatefulSets Caveats
* https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#limitations

## Chart Details
This chart will do the following:

* Implement a dynamically scalable elasticsearch cluster using Kubernetes
StatefulSets/Deployments and also add the kanister blueprint to be used with it.
* Multi-role deployment: master, client (coordinating) and data nodes
* Statefulset Supports scaling down without degrading the cluster

## Installing the Chart

For basic installation, you can install using the provided Helm chart that will
install an instance of Elasticsearch as well as a Kanister blueprint to be used
with it.

Prior to install you will need to have the Elastic Helm repository added to
your local setup.

```bash
$ helm repo add elastic https://helm.elastic.co
$ helm repo update
```

Then install the sample Elasticsearch application with the release name
`elasticsearch` in its own namespace `es-test` using the command below.
Make sure you have the kanister controller running in namespace `kanister`
which is the default setting in Elasticsearch charts. Otherwise, you will also
have to set the `kanister.controller_namespace` parameter value to the
respective kanister controller namespace in the following command:

```bash
$ helm install --namespace es-test elasticsearch elastic/elasticsearch \
  --set antiAffinity=soft --create-namespace
```

The command deploys Elasticsearch on the Kubernetes cluster with the default
configuration.

## Integrating with Kanister

In case, if you don't have `Kanister` installed already, you can use following
commands to do that.
Add Kanister Helm repository and install Kanister operator
```bash
$ helm repo add kanister https://charts.kanister.io
$ helm install kanister --namespace kanister --create-namespace \
  kanister/kanister-operator --set image.tag=0.110.0
```

### Create Profile

Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key> \
  --secret-key <aws-secret-key> --namespace es-test \
  --bucket <s3-bucket-name> --region <region-name>
```

**NOTE:**

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a
`profiles.cr.kanister.io` *CustomResource (CR)* which is then referenced in
Kanister ActionSets. Every ActionSet requires a Profile reference to complete
the action. This CR (`profiles.cr.kanister.io`) can be shared between
Kanister-enabled application instances.

### Create Blueprint

**NOTE: v2 Blueprints are experimental and are not supported with standalone Kanister.**

In order to perform backup, restore, and delete operations on the running
elasticsearch, we need to create a blueprint.

```bash
$ kubectl create -f ./elasticsearch-blueprint.yaml -n kanister
```

Once Elasticsearch is running, you can populate it with some data. Follow the
instructions that get displayed by running command
`helm status elasticsearch -n es-test` to connect to the application.

```bash
# Log in into elasticsearch container and get shell access
$ kubectl exec -it elasticsearch-master-0 -n es-test -c elasticsearch -- bash

# Create index called customer
$ curl -X PUT "https://elastic:${ELASTIC_PASSWORD}@localhost:9200/customer?pretty" -k
{
  "acknowledged" : true,
  "shards_acknowledged" : true,
  "index" : "customer"
}

# Add a customer named John Smith
$ curl -X PUT "https://elastic:${ELASTIC_PASSWORD}@localhost:9200/customer/_doc/1?pretty" \
  -H 'Content-Type: application/json' -d '{"name": "John Smith"}' -k
{
  "_index" : "customer",
  "_id" : "1",
  "_version" : 1,
  "result" : "created",
  "_shards" : {
    "total" : 2,
    "successful" : 2,
    "failed" : 0
  },
  "_seq_no" : 0,
  "_primary_term" : 1
}

# View the data
$ curl -X GET "https://elastic:${ELASTIC_PASSWORD}@localhost:9200/_cat/indices?v" -k
health status index    uuid                   pri rep docs.count docs.deleted store.size pri.store.size
green  open   customer YmIH-p0DRD--KzIA6i4Ayg   1   1          0            0       450b           225b

$ curl -X GET "https://elastic:${ELASTIC_PASSWORD}@localhost:9200/customer/_search?q=*&pretty" -k
{
  "took" : 867,
  "timed_out" : false,
  "_shards" : {
    "total" : 1,
    "successful" : 1,
    "skipped" : 0,
    "failed" : 0
  },
  "hits" : {
    "total" : {
      "value" : 1,
      "relation" : "eq"
    },
    "max_score" : 1.0,
    "hits" : [
      {
        "_index" : "customer",
        "_id" : "1",
        "_score" : 1.0,
        "_source" : {
          "name" : "John Smith"
        }
      }
    ]
  }
}
```

## Protect the Application

You can now take a backup of the Elasticsearch data using an ActionSet defining
backup for this application. Create an ActionSet in the same namespace as the
controller using `kanctl`, a command-line tool that helps create ActionSets as
shown below:

```bash
$ kubectl get profile -n es-test
NAME               AGE
s3-profile-4dxn8   7m25s

$ kanctl create actionset --action backup --namespace kanister \
  --blueprint elasticsearch-blueprint \
  --statefulset es-test/elasticsearch-master \
  --profile es-test/s3-profile-4dxn8
actionset backup-kmms4 created

# View the status of the actionset
$ kubectl --namespace kanister get actionsets.cr.kanister.io
NAME           PROGRESS   LAST TRANSITION TIME   STATE
backup-kmms4   100.00     2023-01-04T09:45:22Z   complete
```

## Disaster strikes!

Let's say someone with fat fingers accidentally deleted the customer index
using the following command:

```bash
# Log in into elasticsearch container and get shell access
$ kubectl exec -it elasticsearch-master-0 -n es-test -c elasticsearch -- bash

# Delete the index
$ curl -X DELETE "https://elastic:${ELASTIC_PASSWORD}@localhost:9200/customer?pretty" -k
{
  "acknowledged" : true
}

# Get the index
$ curl -X GET "https://elastic:${ELASTIC_PASSWORD}@localhost:9200/_cat/indices?v" -k
health status index uuid pri rep docs.count docs.deleted store.size pri.store.size
```

## Restore the Application

To restore the missing data, we want to use the backup created earlier in the
steps above. An easy way to do this is to leverage `kanctl`, a command-line tool
that helps create ActionSets that depend on other ActionSets:

```bash
$ kanctl create actionset --action restore --namespace kanister --from backup-kmms4
actionset restore-backup-kmms4-rp89l created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io restore-backup-kmms4-rp89l
NAME                         PROGRESS   LAST TRANSITION TIME   STATE
restore-backup-kmms4-rp89l   100.00     2023-01-04T09:54:11Z   complete
```

You should now see that the data has been successfully restored to Elasticsearch!

```bash
# Log in into elasticsearch container and get shell access
$ kubectl exec -it elasticsearch-master-0 -n es-test -c elasticsearch -- bash

$ curl -X GET "https://elastic:${ELASTIC_PASSWORD}@localhost:9200/_cat/indices?v" -k
health status index    uuid                   pri rep docs.count docs.deleted store.size pri.store.size
green  open   customer VtP3QddrTdq69mvq3NwCuQ   1   1          1            0      9.2kb          4.5kb

$ curl -X GET "https://elastic:${ELASTIC_PASSWORD}@localhost:9200/customer/_search?q=*&pretty" -k
{
  "took" : 34,
  "timed_out" : false,
  "_shards" : {
    "total" : 1,
    "successful" : 1,
    "skipped" : 0,
    "failed" : 0
  },
  "hits" : {
    "total" : {
      "value" : 1,
      "relation" : "eq"
    },
    "max_score" : 1.0,
    "hits" : [
      {
        "_index" : "customer",
        "_id" : "1",
        "_score" : 1.0,
        "_source" : {
          "name" : "John Smith"
        }
      }
    ]
  }
}
```

## Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the
following command:

```bash
$ kanctl create actionset --action delete --namespace kanister \
  --from backup-kmms4 --namespacetargets kanister
actionset delete-backup-kmms4-sd6tj created

# View the status of the ActionSet
$ kubectl --namespace kanister get actionsets.cr.kanister.io delete-backup-kmms4-sd6tj
NAME                        PROGRESS   LAST TRANSITION TIME   STATE
delete-backup-kmms4-sd6tj   100.00     2023-01-04T09:59:53Z   complete
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of
the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```

## Cleanup

### Uninstalling the chart

```bash
$ helm delete elasticsearch -n es-test
release "elasticsearch" uninstalled
```

Deletion of the StatefulSet doesn't cascade to deleting associated PVCs.
To delete them:

```bash
$ kubectl delete pvc -l app=elasticsearch-master -n es-test
persistentvolumeclaim "elasticsearch-master-elasticsearch-master-0" deleted
persistentvolumeclaim "elasticsearch-master-elasticsearch-master-1" deleted
persistentvolumeclaim "elasticsearch-master-elasticsearch-master-2" deleted
```

### Delete CRs

Remove Blueprint, Profile CR and ActionSet

```bash
$ kubectl delete blueprints.cr.kanister.io elasticsearch-blueprint -n kanister
blueprint.cr.kanister.io "elasticsearch-blueprint" deleted

$ kubectl get profiles.cr.kanister.io -n es-test
NAME               AGE
s3-profile-4dxn8   93m

$ kubectl delete profiles.cr.kanister.io s3-profile-4dxn8 -n es-test
profile.cr.kanister.io "s3-profile-4dxn8" deleted

$ kubectl delete actionset backup-kmms4 delete-backup-kmms4-sd6tj \
  restore-backup-kmms4-rp89l -n kanister
actionset.cr.kanister.io "backup-kmms4" deleted
actionset.cr.kanister.io "delete-backup-kmms4-sd6tj" deleted
actionset.cr.kanister.io "restore-backup-kmms4-rp89l" deleted
```

## Configuration

If you're on a single node cluster, you'd need to set the antiAffinity to soft
while installing the helm chart by running `--set antiAffinity=soft` so that
pods are not stuck in the pending state. For other configurations of
elasticsearch helm chart, please refer
https://github.com/elastic/helm-charts/blob/master/elasticsearch/README.md
