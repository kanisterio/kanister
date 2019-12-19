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
fdb> set lastname Manville
```

# Protect the application

Once we have the foundationDB cluster up and running we can go ahead and integrate
this with Kanister in order to take the backup and restore that backup.

## Create profile 

```bash
kanctl create profile s3compliant --access-key <access-key>                 \
                --secret-key <secret-key>                                   \
                --bucket infracloud.kanister.io --region <region-name>      \
                --namespace default
profile s3-profile-8fs88 created

```

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.


## Create blueprint

In order to perform backup, restore, and delete operations on the running foundationDB,
we need to create a Blueprint.
You can create the Blueprint using the command below.

```bash
kubectl create -f foundationdb-blueprint.yaml -n kanister
```

Once we have created the Blueprint let's go ahead and insert some data into the foundationDB
database.

### Insert some records
To insert some records into the database we will have to `EXEC` into the foundationDB pod
and then run the database command in the `fdbcli` utility

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
using the CR, we will have create a `ClusterRoleBinding` to enable that access. You can create that
resource using below command

```bash
$ kubectl create -f kanister-clusteradmin-binding.yaml 
```


## Create backup actionset 

To take the backup of the data tha we have just inserted into the database, we will have to create 
Actionset Kanister resource . Please follow below command to create the Actionset

```bash
kanctl create actionset --action backup --namespace kanister --blueprint foundationdb-blueprint  
    --profile default/s3-profile-8fs88  \
    --objects apps.foundationdb.org/v1beta1/foundationdbclusters/default/foundationdbcluster-sample

actionset backup-jx2d2 created
```
Once you have created the Actionset, you can check the status of the Actionset to make sure the backup
is completed.

## Disaster strikes!
Once the backup is completed we can go ahead and delete the data manually from the database to imitate
disaster.

```bash
$ kubectl exec -it foundationdbcluster-sample-1 -c foundationdb bash
$ fdbcli
fdb> writemode on
fdb> clearrange '' \xFF
```

Once you cleared all the keys, we can go ahead to get the value of any key that we have inserted and
we should not get the value  of that key.

```bash
$ kubectl exec -it foundationdbcluster-sample-1 -c foundationdb bash
$ fdbcli
fdb> get name
`name': not found
```

Once we have deleted the data from the database, to imitate the disaster, we will restore the backup that
we have already created. To restore the backup we will have to create another Actionset, with `restore` action.

```bash
$ kanctl --namespace kanister create actionset --action restore --from "backup-jx2d2"
actionset <restore-actionset> created
```
Once you have created the restore actionset, you can make sure that the actionset is completed by describing the
actionset. You can describe the actionset using below command

```bash

$ kubectl describe actionset -n <kanister-operator-namespace> <restore-actionset-name>
```

## Delete the artifacts
The artifacts created by the backup action can be cleaned up using the following command:

```bash
$ kanctl --namespace kanister create actionset --action delete --from "backup-jx2d2"
actionset "<delete-actionset-name>" created

# View the status of the ActionSet
$ kubectl --namespace kanister describe actionset <delete-actionset-name>

```

## Troubleshooting
If you run into any issues with the above commands, you can check the logs of the controller using:

```bash
$ kubectl --namespace kanister logs -l app=kanister-operator
```
you can also check events of the actionset

```bash
$ kubectl describe actionset <actionset-name> -n kanister
```

# Delete Blueprint and Profile CR

```bash
$ kubectl delete blueprints.cr.kanister.io foundationdb-blueprint -n kanister

$ kubectl get profiles.cr.kanister.io -n default
NAME               AGE
s3-profile-sph7s   2h

$ kubectl delete profiles.cr.kanister.io s3-profile-sph7s -n default
```