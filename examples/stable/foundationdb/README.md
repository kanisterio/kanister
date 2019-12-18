# FoundationDB

According to their documentation FoundationDB gives us the power of ACID
transactions in a distributed database.

**Note**
This README is highly inspired by the steps that are mentioned in the
repository of [foundationDB operator](https://github.com/foundationdb/fdb-kubernetes-operator).
Since they don't have any any way to install the operator without
building the operator image, we have followed the the upstream's `Makefile`
to create this README that can be used to create a simple foundationDB
cluster.


## Prerequisite 

* You should have [Kustomize](https://github.com/kubernetes-sigs/kustomize) installed
on you cluster.
* Kubernetes 1.9+ with Beta APIs enabled.
* PV support on the underlying infrastructure.
* Kanister version 0.23.0 with `profiles.cr.kanister.io` CRD installed

# Installation

We don't have the foundationDB chart yet to get the foundationDB installed
on any Kubernetes cluster. So we will be manually installing foundationDB
on our Kubernetes cluster.
FoundationDB team recently started working on
[an operator](https://github.com/foundationdb/fdb-kubernetes-operator),
and we wil be using operator to install foundationDB cluster on our Kubernetes
cluster.

To get to know about how Kubernetes operators, please follow
[this link](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

### Created Secrets to set up a secret with self-signed test certs

To create the secrets please follow below command

```bash
$ ./config/test-certs/generate_secrets.bash
```

### Deploy the Operator 

Deploy the `FoundationDBCluster` CRD and then we are using `Kustomize`
and `kubectl` to deploy other supporting resources of this operator 

```bash
kubectl apply -f crds/apps_v1beta1_foundationdbcluster.yaml

kustomize build config/default | kubectl apply -f -
```


### Create the cluster

Once we have the CRD `FoundationDBCluster` created we can go ahead with
creating the CR for the already created CRD that will spin up a foundationDB
cluster.

Please follow below command to create the CR

```bash
$ kubectl create -f local_cluster.yaml
```

If you now go ahead and try to list all the pods, you will be able to see
that there are some pods running to have foundationDB cluster. 
By default we are running this cluster in double
[redundancy mode](https://apple.github.io/foundationdb/configuration.html#choosing-a-redundancy-mode),
because we faced some issues while running it as single redundancy mode, and
have discussion going on about that, [here](https://forums.foundationdb.org/t/connecting-to-the-database-using-fdbcli-results-in-an-error/1841).

Let's try to exec into a pod and then try to insert some key value pairs 
in the database 

```bash
# exec into the foundation db pod
$ kubectl exec -it foundationdbcluster-sample-1 -c foundationdb bash

# once you are in the foundation DB pod get the fdbcli to run the command
$ fdbcli

fdb> writemode on 
fdb> set name Tom
fdb> set lastname Singh
```

**Note**
Steps that are mentioned below are still WIP.

Once we have the foundationDB cluster up and running we can go ahead and integrate
this with Kanister in order to take the backup and restore that backup.

## Create profile 

```bash
kanctl create profile s3compliant --access-key <access-key>                 \
                --secret-key <secret-key>                                   \
                --bucket infracloud.kanister.io --region <region-name>      \
                --namespace default

```

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.


## Create blueprint

In order to perform backup, restore, and delete operations on the running foundationDB,
we need to create a blueprint.
You can create that using the command below.

```bash
kubectl create -f foundationdb-blueprint.yaml -n kanister
```


### Insert some records


## Create backup actionset 

```bash
kanctl create actionset --action backup --namespace kanister --blueprint foundationdb-blueprint  --profile default/s3-profile-wrv4r --statefulset default/fdb-kubernetes-operator-controller-manager

actionset backup-jx2d2 created

```

### Delete the records that we have inserted

