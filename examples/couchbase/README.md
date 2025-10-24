# Couchbase

[Couchbase](https://www.couchbase.com) is an open-source, distributed multi-model NoSQL document-oriented database software package that is optimized for interactive applications. Couchbase Server is designed to provide easy-to-scale key-value or JSON document access with low latency and high sustained throughput. It is designed to be clustered from a single machine to very large-scale deployments spanning many machines.

## Prerequisites

* Kubernetes 1.20+
* PV support on the underlying infrastructure
* Kanister controller version 0.22.0 installed in your cluster
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)
* Couchbase application needs at least 4G of memory and 4 cpus to run. Make sure that your cluster nodes has required resources.
* Docker CLI installed

### Build and push couchbase-tools docker image

Kanister actions are specified in the Blueprint CR. The example Blueprint for the Couchbase application uses the `couchbase-tools` docker image to perform required actions. Follow the steps below to build the docker image with all the necessary tools.

```bash
# Clone Kanister Github repo locally
$ git clone https://github.com/kanisterio/kanister.git <path_to_kanister>

# Build couchbase-tools docker image
$ docker build -t <registry>/<repository>/couchbase-tools:<tag_name> <path_to_kanister>/docker/couchbase-tools/
$ docker push <registry>/<repository>/couchbase-tools:<tag_name>
```

Replace `<registry>`, `<repository>` and `<tag_name>` placeholders with actual values in `couchbase-blueprint.yaml`.

## Chart Details

We will be using [couchbase](https://github.com/couchbase-partners/helm-charts) official helm charts to deploy [Couchbase Operator](https://docs.couchbase.com/operator/current/overview.html) and Couchbase cluster on [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Installing the Chart

To install the chart with the release name `cb-example`:

```bash
#NOTE: The latest couchbase helm charts require Helm 3.1+
$ helm repo add couchbase https://couchbase-partners.github.io/helm-charts
$ helm repo update
# Create couchbase-test namespace
$ kubectl create ns couchbase-test
# The below chart installs couchbase cluster, admission controller and operator
$ helm install cb-example couchbase/couchbase-operator --namespace couchbase-test \
    --set cluster.servers.default.size=2 \
    --set cluster.servers.default.services[0]=data \
    --set cluster.servers.default.services[1]=query \
    --set cluster.servers.default.services[2]=index
```

**NOTE:**

Kanister operator roles needs to be updated in order to access custom resource.
For testing purpose, we can give cluster-admin role to kanister-operator service account with -

`kubectl create -f kanister-clusteradmin-binding.yaml`

## Integrating with Kanister

If you have deployed couchbase operator with other name than `cb-example` and namespace other than `couchbase-test`, you need to modify the commands used below to use the correct name and namespace

### Create Profile

Create Profile CR if not created already

```bash
$ kanctl create profile s3compliant --access-key <aws-access-key-id> \
	--secret-key <aws-secret-key> \
	--bucket <s3-bucket-name> --region <region-name> \
	--namespace couchbase-test
```

**NOTE:**

The command will configure a location where artifacts resulting from Kanister data operations such as backup should go. This is stored as a profiles.cr.kanister.io CustomResource (CR) which is then referenced in Kanister ActionSets. Every ActionSet requires a Profile reference to complete the action. This CR (profiles.cr.kanister.io) can be shared between Kanister-enabled application instances.


### Create Blueprint

**NOTE: v2 Blueprints are experimental and are not supported with standalone Kanister.**

Create Blueprint in the same namespace as the controller

```bash
$ kubectl create -f couchbase-blueprint.yaml -n kasten-io
blueprint.cr.kanister.io/couchbase-blueprint created
```

Once Couchbase is running, you can populate it with some data. Let's add some documents to the default bucket:

```bash
# Connect to couchbase cluster pod
$ kubectl exec -ti cb-example-couchbase-cluster-0000 -n couchbase-test -- bash

# From inside the shell, use the cbc command to insert some data into the default bucket
# Replace <password> with the correct password.
$ cbc-create -u Administrator -P <password>  doc1 -V '{"name":"vivek", "age": 25}'
$ cbc-create -u Administrator -P <password>  doc2 -V '{"name":"prasad", "age": 25}'

# verify data
$ cbc-n1ql -u Administrator -P <password> 'create primary index on default'
$ cbc-n1ql -u Administrator -P <password> 'select * from default'
---> Encoded query: {"statement":"select * from default"}

{"default":{"name":"prasad", "age": 25}},
{"default":{"name":"vivek", "age": 25}},
---> Query response finished
{
"requestID": "46178b60-8297-43f3-a94e-19a6f9207f98",
"signature": {"*":"*"},
"results": [
],
"status": "success",
"metrics": {"elapsedTime": "14.711966ms","executionTime": "14.510259ms","resultCount": 1,"resultSize": 40}
}
```

## Protect the Application

You can now take a backup of the Couchbase cluster data using an ActionSet defining backup for this application. Create an ActionSet in the same namespace as the controller.

```bash
$ kubectl get profile -n couchbase-test
NAME               AGE
s3-profile-bghqw   2m42s

$ kanctl create actionset --action backup --namespace kasten-io --blueprint couchbase-blueprint --profile couchbase-test/s3-profile-bghqw --objects couchbase.com/v2/couchbaseclusters/couchbase-test/cb-example-couchbase-cluster
actionset backup-g648n created

$ kubectl --namespace kasten-io get actionsets.cr.kanister.io
NAME           AGE
backup-g648n   37s

# View the status of the actionset
$ kubectl --namespace kasten-io describe actionset backup-g648n
```

### Disaster strikes!

Let's say someone with accidentally deleted the bucket using the following command in couchbase cluster pod:

```bash
# Drop default bucket
# Replace <password> with the correct password.
$ cbc-bucket-delete -u Administrator -P <password> default
Requesting /pools/default/buckets/default
200
  Cache-Control: no-cache,no-store,must-revalidate
  Content-Length: 0
  Date: Thu, 12 Dec 2019 12:01:06 GMT
  Expires: Thu, 01 Jan 1970 00:00:00 GMT
  Pragma: no-cache
  Server: Couchbase Server
  X-Content-Type-Options: nosniff
  X-Frame-Options: DENY
  X-Permitted-Cross-Domain-Policies: none
  X-XSS-Protection: 1; mode=block
```

### Restore the Application

To restore the missing data, you should use the backup that you created before. An easy way to do this is to leverage `kanctl`, a command-line tool that helps create ActionSets that depend on other ActionSets:

```bash
$ kanctl --namespace kasten-io create actionset --action restore --from "backup-g648n"
actionset restore-backup-g648n-64gqm created

# View the status of the ActionSet
kubectl --namespace kasten-io describe actionset restore-backup-g648n-64gqm
```

You should now see that the data has been successfully restored to Couchbase cluster!

```bash
# Recreate index
# Replace <password> with the correct password.
$ cbc-n1ql -u Administrator -P <password> 'drop primary index on default'
$ cbc-n1ql -u Administrator -P <password> 'create primary index on default'
$ cbc-n1ql -u Administrator -P <password> 'select * from default'
---> Encoded query: {"statement":"select * from default"}

{"default":{"name":"vivek", "age": 25}},
{"default":{"name":"prasad", "age": 25}},
---> Query response finished
{
"requestID": "8325812b-5c7b-4848-a59e-82d08ba11191",
"signature": {"*":"*"},
"results": [
],
"status": "success",
"metrics": {"elapsedTime": "18.153378ms","executionTime": "17.972551ms","resultCount": 2,"resultSize": 79}
}
```

### Delete the Artifacts

The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kasten-io create actionset --action delete --from "backup-g648n" --namespacetargets kasten-io
actionset "delete-backup-g648n-k9ncm" created

# View the status of the ActionSet
$ kubectl --namespace kasten-io describe actionset delete-backup-g648n-k9ncm
```

### Troubleshooting

If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kasten-io logs -l app=kanister-operator
```

you can also check events of the actionset

```bash
$ kubectl describe actionset restore-backup-g648n-64gqm -n kasten-io
```

## Uninstalling the Chart

Use helm delete command to remove couchbase-operator and couchbase-cluster

```bash
$ helm delete cb-example -n couchbase-test
$ helm delete couchbase-operator -n couchbase-test
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

Delete Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io couchbase-blueprint -n kasten-io

$ kubectl get profiles.cr.kanister.io -n couchbase-test
NAME               AGE
s3-profile-sph7s   2h

$ kubectl delete profiles.cr.kanister.io s3-profile-sph7s -n couchbase-test
```
