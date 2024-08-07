# FoundationDB

According to the foundationDB documentation FoundationDB gives us the power of ACID
transactions in a distributed database and has below properties

* Multi-model data store
* Easily scalable and fault tolerant
* Industry-leading performance

**Note**
These steps are taken from the steps that are mentioned in the official
repository of [foundationDB operator](https://github.com/foundationdb/fdb-kubernetes-operator).
Since they don't have any any way to install the operator without
building the operator image, we have followed the the upstream `Makefile`
to create this README that can be used to create a simple foundationDB
cluster.


## Prerequisite

* Install GO on your machine, see the [Getting Started](https://golang.org/doc/install) guide for more information, and [set GOPATH](https://github.com/golang/go/wiki/SettingGOPATH).
* Install KubeBuilder and its dependencies on your machine, see [The KubeBuilder Book](https://book.kubebuilder.io/quick-start.html) for more information.
* You should have [Kustomize](https://github.com/kubernetes-sigs/kustomize) installed
on you cluster.
* Kubernetes 1.9+ with Beta APIs enabled.
* PV support on the underlying infrastructure.
* Kanister version 0.110.0 with `profiles.cr.kanister.io` CRD installed.
* Docker CLI installed
* A docker image containing the required tools to back up FoundationDB.
The Dockerfile for the image can be found [here](https://raw.githubusercontent.com/kanisterio/kanister/master/docker/foundationdb/Dockerfile).
To build and push the docker image to your registry, execute [these](#build-docker-image) steps.

### Build docker image

Execute the commands below to build and push the `foundationdb` docker image to a registry.
```bash
# Clone Kanister Github repo locally
$ git clone https://github.com/kanisterio/kanister.git <path_to_kanister>

# Build FoundationDB docker image
$ docker build -t <registry>/<repository>/foundationdb:<tag_name> <path_to_kanister>/docker/foundationdb
$ docker push <registry>/<repository>/foundationdb:<tag_name>
```

# Installation

We don't have the foundationDB helm chart yet to get the foundationDB installed
on our Kubernetes cluster. So we will be manually installing foundationDB
on our Kubernetes cluster.
FoundationDB team recently started working on
[an operator](https://github.com/foundationdb/fdb-kubernetes-operator),
and we wil be using operator to install foundationDB cluster on our Kubernetes
cluster.

To get to know about how Kubernetes operators, please follow
[this link](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

Steps that are mentioned below can be followed to install the foundationDB operator

* Change your current directory to `$GOPATH/src/github.com` using the command `cd $GOPATH/src/github.com`
and run `mkdir foundationdb` to create the directory `foundationdb`.
* `CD` into newly created directory and clone this github repo inside the created directory that is `foundationdb`
using below command `git clone https://github.com/FoundationDB/fdb-kubernetes-operator.git`.
* Run `sed -i '/IMG ?= fdb-kubernetes-operator:latest/c\IMG ?= ghcr.io/kanisterio/fdb-kubernetes-operator:latest' Makefile` to correct the name of the operator image.
* Create Secrets to set up a secret with self-signed test certs using the command `./config/test-certs/generate_secrets.bash`
* Run `make rebuild-operator` to install the operator. Please make sure this completes successfully.


### Create the cluster

Once we have the CRD `FoundationDBCluster` (this will be taken care by `make rebuild-operator`) created we
can go ahead with creating the CR for the already created CRD that will spin up a foundationDB
cluster.

Please follow below command to create the CR, please make sure to use the local_cluster.yaml file from this repo.

**NOTE:**

Replace `<registry>`, `<repository>` and `<tag_name>` for the `imageName` value in `./local_cluster.yaml` before running the following command.

```bash
$ kubectl create -f local_cluster.yaml
```

If you now go ahead and try to list all the pods, you will be able to see
that there are some pods running for foundationDB cluster.
By default we are running this cluster in double
[redundancy mode](https://apple.github.io/foundationdb/configuration.html#choosing-a-redundancy-mode),
because we faced some issues while running it as single redundancy mode, and
have discussion going on about that, [here](https://forums.foundationdb.org/t/connecting-to-the-database-using-fdbcli-results-in-an-error/1841).

Once we have the database pods up and running let's try to exec into a pod and
then try to insert some key value pairs  in the database

```bash
# exec into the foundation db pod
$ kubectl exec -it foundationdbcluster-sample-1 -c foundationdb bash

# once you are in the foundation DB pod get the fdbcli to run the command
$ fdbcli

fdb> writemode on
fdb> set name Tom
fdb> set lastname Manville
```

# Protect the application

Once we have the foundationDB cluster up and running we can go ahead and integrate
this with Kanister in order to take the backup and restore that backup. To achieve
that we will have to create below mentioned Kanister resources

## Create profile

```bash
kanctl create profile s3compliant --access-key <access-key>                 \
                --secret-key <secret-key>                                   \
                --bucket infracloud.kanister.io --region <region-name>      \
                --namespace default
profile <profile-name> created

```

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.


## Create blueprint

**NOTE: v2 Blueprints are experimental and are not supported with standalone Kanister.**

In order to perform `backup`, `restore`, and `delete` operations on the running foundationDB,
we need to create a Blueprint Kanister resource.
You can create the Blueprint using the command below.

```bash
# replace kanister-op-ns with the namespace where kanister is installed
kubectl create -f foundationdb-blueprint.yaml -n <kanister-op-ns>
```

Once we have created the Blueprint let's go ahead and insert some data into the foundationDB
database.

### Insert some records
To insert some records into the database we will have to `EXEC` into the foundationDB pod
and then run the database command using the `fdbcli` utility

```bash
# EXEC into the pod
$ kubectl exec -it foundationdbcluster-sample-1 -c foundationdb bash
# get the fdbcli
$ fdbcli
fdb> writemode on
fdb> set name Tom
fdb> set companyname Kasten
fdb> set lasltname Manville
```
# Create ClusterRoleBinding
Since the Kanister resources need access to the the foundationDB cluster that has just been created,
using the CR, we will have create a `ClusterRoleBinding` to enable that access. You can create that `ClusterRoleBinding`
resource using below command

```bash
$ kubectl create -f kanister-clusteradmin-binding.yaml
```


## Create backup actionset

To take the backup of the data tha we have just inserted into the database, we will have to create
Actionset Kanister resource. Please follow below command to create the Actionset

```bash
$ kanctl create actionset --action backup --namespace <kanister-op-ns> --blueprint foundationdb-blueprint  \
    --profile default/<profile-name>  \
    --objects apps.foundationdb.org/v1beta1/foundationdbclusters/default/foundationdbcluster-sample

actionset backup-jx2d2 created
```
Once you have created the Actionset, you can check the status of the Actionset by describing it to make sure
the backup is completed.
Please follow below command to check the status of the Actionset

```bash
$ kubectl describe actionset -n <kanister-op-ns> backup-jx2d2
```

## Disaster strikes!
Once the backup is completed we can go ahead and delete the data manually from the database to imitate
disaster.

```bash
$ kubectl exec -it foundationdbcluster-sample-1 -c foundationdb bash
$ fdbcli
fdb> writemode on
fdb> clearrange '' \xFF
```

Once we have cleared all the keys, we can go ahead and try get the value of any key that we have inserted and
we should not get the value of that key because we have deleted the data.

```bash
$ kubectl exec -it foundationdbcluster-sample-1 -c foundationdb bash
$ fdbcli
fdb> get name
`name': not found
```

Once we have deleted the data from the database, to imitate the disaster, we will restore the backup that
we have already created. To restore the backup we will have to create another Actionset, with `restore` action.

```bash
# Please replace kanister-op-ns with namespace where your kanister is installed
$ kanctl --namespace <kanister-op-ns> create actionset --action restore --from "backup-jx2d2"
actionset <restore-actionset> created
```
Once you have created the restore actionset, you can make sure that the actionset is completed by describing the
actionset. You can describe the actionset using below command

```bash

$ kubectl describe actionset -n <kanister-operator-namespace> <restore-actionset-name>
```
If you have verified that the `restore` actionset is completed we can `EXEC` into the foundationDB pod once again
to verify if the data has been restored back.


## Delete the artifacts
The artifacts created by the backup action can be cleaned up using the following command:

```bash
# Replace kanister-op-ns with the namespace where your kanister operator is installed
$ kanctl --namespace <kanister-op-ns> create actionset --action delete --from "backup-jx2d2"
actionset "<delete-actionset-name>" created

# View the status of the ActionSet
# Replace kanister-op-ns with the namespace where your kanister operator is installed
$ kubectl --namespace <kanister-op-ns> describe actionset <delete-actionset-name>

```

## Troubleshooting
If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
# Replace kanister-op-ns with the namespace where your kanister operator is installed
$ kubectl --namespace <kanister-op-ns> logs -l app=kanister-operator
```
you can also check events of the actionset

```bash
$ kubectl describe actionset <actionset-name> -n kanister
```

# Delete Blueprint and Profile CR

```bash
# Replace kanister-op-ns with the namespace where your kanister operator is installed
$ kubectl delete blueprints.cr.kanister.io foundationdb-blueprint -n <kanister-op-ns>

$ kubectl get profiles.cr.kanister.io -n default
NAME               AGE
s3-profile-sph7s   2h

$ kubectl delete profiles.cr.kanister.io s3-profile-sph7s -n default
```
