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
* Kanister version 0.28.0 with `profiles.cr.kanister.io` CRD installed

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
$ helm repo add elastic https://helm.elastic.co
```

Then install the sample Elasticsearch application with the release name `my-release` in its own namespace
`es-test` using the command below. Make sure you have the kanister controller running in namespace `kasten-io` which is the default setting in Elasticsearch charts. Otherwise, you will also have to set the `kanister.controller_namespace` parameter value to the respective kanister controller namespace in the following command:

```bash
$ helm install --namespace es-test --name elasticsearch elastic/elasticsearch --set antiAffinity=soft
```
If you are running helm version `v3.0.0`, please use the commands below:
```bash
$ kubectl create namespace es-test
$ helm install --namespace es-test elasticsearch elastic/elasticsearch --set antiAffinity=soft
```

The command deploys Elasticsearch on the Kubernetes cluster in the default
configuration.

```bash
kanctl --namespace es-test create profile --bucket <bucket-name> --region ap-south-1 s3compliant --access-key <aws-access-key> --secret-key <aws-secret-key>
```
This command creates a profile which we will use later.

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.

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



$ curl 'localhost:9200/customer/_search?q=*&pretty'

{
  "took" : 9,
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
        "_type" : "_doc",
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

## Create the Blueprint

In order to perform backup, restore, and delete operations on the running elasticsearch, we need to create a blueprint. You can create that using the command below from the root of kanister repo.

```bash
kubectl create -f ./examples/stable/elasticsearch/elasticsearch-blueprint.yaml -n kasten-io
```

## Protect the Application

You can now take a backup of the Elasticsearch data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller using `kanctl`, a command-line tool that helps create ActionSets as shown below:

```bash
$ kanctl create actionset --action backup --namespace kasten-io --blueprint elasticsearch-blueprint --statefulset es-test/elasticsearch-master --profile es-test/<PROFILE_NAME>

$ kubectl --namespace kasten-io get actionsets.cr.kanister.io
NAME                AGE
backup-lphk7        2h

# View the status of the actionset
$ kubectl --namespace kasten-io describe actionset backup-lphk7
```

The PROFILE_NAME is the name of the profile generated from earlier kanctl create profile command. It can be retrieved using,

```bash
kubectl get profiles.cr.kanister.io -n es-test
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

If you're on a single node cluster, you'd need to set the antiAffinity to soft while installing the helm chart by running `--set antiAffinity=soft` so that pods are not stuck in the pending state. For other configurations of elasticsearch helm chart, please refer https://github.com/elastic/helm-charts/blob/master/elasticsearch/README.md
