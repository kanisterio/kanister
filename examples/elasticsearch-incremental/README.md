# Elasticsearch

[Elasticsearch](https://www.elastic.co/) is a search engine based 
on the Lucene library. It provides a distributed, multitenant-capable full-text
 search engine with an HTTP web interface and schema-free JSON documents.

This blueprint in this example uses the 
[elasticsearch snapshot api](https://www.elastic.co/guide/en/elasticsearch/reference/current/snapshot-restore.html) 
to protect your cluster. This snapshot is incremental and the blueprint works 
only for 
[s3 compatible repository](https://www.elastic.co/guide/en/elasticsearch/reference/current/repository-s3.html).

## Prerequisites

* Kubernetes 1.9+
* PV support on the underlying infrastructure
* Kanister controller version 0.81.0 or above installed in your cluster
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

## Operator Details

We will be using the 
[elasticsearch operator](https://www.elastic.co/guide/en/cloud-on-k8s/current/index.html) 
to deploy the elasticsearch cluster.

## Installing the Operator and the cluster

Install the operator
```
kubectl create -f https://download.elastic.co/downloads/eck/2.4.0/crds.yaml
kubectl apply -f https://download.elastic.co/downloads/eck/2.4.0/operator.yaml
```

Monitor the logs 
```
kubectl -n elastic-system logs -f statefulset.apps/elastic-operator
```

Create an Elastic search cluster in `test-es1` namespace by creating `Elasticsearch` resource
```
kubectl create ns test-es1
kubectl config set-context --current --namespace test-es1
cat <<EOF | kubectl apply -f -
apiVersion: elasticsearch.k8s.elastic.co/v1
kind: Elasticsearch
metadata:
  name: quickstart
spec:
  version: 8.4.1
  nodeSets:
  - name: default
    count: 2     
    volumeClaimTemplates:
    - metadata:
        name: elasticsearch-data # Do not change this name unless you set up a volume mount for the data path.
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 100Gi  
    config:
      node.store.allow_mmap: false
EOF
```

## Integrating with Kanister

If you have deployed the elasticsearch cluster with other name 
than `quickstart` and namespace other than `test-es1`, you need
to modify the commands used below to use the correct name and namespace

### Create Profile

Create Profile CR, if not created already, using the below command

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--bucket <s3-bucket-name> --region <region-name> \
	--namespace test-es1
```

**NOTE:**

The command will configure a location where artifacts 
resulting from Kanister data operations such as backup 
should go. This is stored as a profiles.cr.kanister.io 
CustomResource (CR) which is then referenced in Kanister 
ActionSets. Every ActionSet requires a Profile reference
to complete the action. This CR (profiles.cr.kanister.io) 
can be shared between Kanister-enabled application instances.


### Create Blueprint

Create Blueprint in the same namespace as the Kanister controller

```bash
$ kubectl create -f elasticsearch-incremental-blueprint.yaml -n kasten-io
```

Once Elasticsearch is running, you can populate it with some data. 

Create a client pod that has `curl` utility in it: 

```
PASSWORD=$(kubectl get -n test-es1 secret quickstart-es-elastic-user -o go-template='{{.data.elastic | base64decode}}')
ES_URL="https://quickstart-es-http:9200"
kubectl run -n test-es1 curl -it --restart=Never --rm --image ghcr.io/kanisterio/kanister-kubectl-1.18:0.81.0 --env="PASSWORD=$PASSWORD" --env="ES_URL=$ES_URL" --command bash 
```

In the curl pod shell
```
curl -u "elastic:$PASSWORD" -k "${ES_URL}"
# List the index 
curl  -k -u "elastic:$PASSWORD" -X GET "${ES_URL}/*?pretty"
# Create an index 
curl  -k -u "elastic:$PASSWORD" -X PUT "${ES_URL}/my-index-000001?pretty"
#  add an index and a document 
curl  -k -u "elastic:$PASSWORD" -X PUT "${ES_URL}/my-index-000002/_doc/1?timeout=5m&pretty" -H 'Content-Type: application/json' -d'
{
  "@timestamp": "2099-11-15T13:12:00",
  "message": "GET /search HTTP/1.1 200 1070000",
  "user": {
    "id": "kimchy-2"
  }
}
'
# retreive document 
curl -k -u "elastic:$PASSWORD" -X GET "${ES_URL}/my-index-000002/_doc/1?pretty"
```


## Protect the Application

You can now take a backup of the elasticsearch data 
using an ActionSet defining backup for this application. 
Create an ActionSet in the same namespace as the Kanister controller.

```bash
$ kubectl get profile -n test-es1
NAME               AGE
s3-profile-sph7s   2h

$ kanctl create actionset --action backup --namespace kasten-io \
  --blueprint elasticsearch-incremental-blueprint \
	--objects elasticsearch.k8s.elastic.co/v1/elasticsearches/test-es1/quickstart \
	--profile test-es1/s3-profile-sph7s
actionset backup-llfb8 created

$ kubectl --namespace kasten-io get actionsets.cr.kanister.io
NAME                 AGE
backup-llfb8         2h

# View the status of the actionset
$ kubectl --namespace kasten-io describe actionset backup-llfb8
```

### Disaster strikes!

Let's say someone accidentally deleted all the indices you created 
```bash
curl -k -u "elastic:$PASSWORD" -X DELETE "${ES_URL}/my-index-000001?pretty"
curl -k -u "elastic:$PASSWORD" -X DELETE "${ES_URL}/my-index-000002?pretty"
```

If you try to access this data in the database, you should see a 404 error:
```bash
curl -k -u "elastic:$PASSWORD" -X GET "${ES_URL}/my-index-000002/_doc/1?pretty"
```

### Restore the Application

To restore the missing data, you should use the backup that you 
created before. An easy way to do this is to leverage `kanctl`, 
a command-line tool that helps create ActionSets that depend 
on other ActionSets:


```bash
$ kanctl --namespace kasten-io create actionset --action restore --from "backup-llfb8"
actionset restore-backup-llfb8-64gqm created

# View the status of the ActionSet
kubectl --namespace kasten-io describe actionset restore-backup-llfb8-64gqm
```

You should now see that the data has been successfully restored to MongoDB!

```bash
curl -k -u "elastic:$PASSWORD" -X GET "${ES_URL}/my-index-000002/_doc/1?pretty"
```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the 
following command:

```bash
$ kanctl --namespace kasten-io create actionset --action delete --from "backup-llfb8"
actionset "delete-backup-llfb8-k9ncm" created

# View the status of the ActionSet
$ kubectl --namespace kasten-io describe actionset delete-backup-llfb8-k9ncm
```

### Troubleshooting

If you run into any issues with the above commands, you can check 
the logs of the controller using:

```bash
$ kubectl --namespace kasten-io logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-llfb8-64gqm -n kasten-io
```

If you want to get only the logs created by the Kubetask function you can 
- Temporary replace `- +x` bash option by `- -x` in the blueprint
- Use this command to get only the output of the kubetask pod `kubectl logs -n kasten-io -l component=kanister --tail=10000 -f | ggrep '"LogKind":"datapath"' | ggrep -o -P '(?<="Pod_Out":").*?(?=",)'`

## Uninstalling the elasticsearch cluster

To uninstall/delete the `quickstart` cluster:

```bash
$ kubectl delete elasticsearch quickstart -n test-es1
```

The command removes all the Kubernetes components associated
 with the elasticsearch cluster quickstart and deletes the 
 quickstart object itself.

Delete Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io elasticsearch-incremental-blueprint -n kasten-io

$ kubectl get profiles.cr.kanister.io -n test-es1
NAME               AGE
s3-profile-sph7s   2h

$ kubectl delete profiles.cr.kanister.io s3-profile-sph7s -n test-es1
```
