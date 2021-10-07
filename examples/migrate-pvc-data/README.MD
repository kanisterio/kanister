# Migrate the volume data of a PVC from a kubernetes cluster to another one 

We'll demonstrate how to migrate the volume data of a PVC from a cluster (source cluster )
to another one (destination cluster) with Kanister.

This example was tested to migrate a PVC from an openshift 3.9 cluster to an openshift 4.7 cluster.
It's for this reason we install kanister in version 0.61.0 because it's the last kanister version still 
compatible with openshift 3.9. If you don't have this constraint you can install kanister in the latest
version.

## Requirement 

The PVC must not be mounted by any pods during the backup or during the restore.

## Prerequisites on both source and destination cluster

* Kubernetes 1.9+
* Kubernetes beta APIs enabled only if `podDisruptionBudget` is enabled
* PV support on the underlying infrastructure
* Kanister controller version 0.61.0 installed in your cluster, let's say in namespace `<kanister-operator-namespace>`
* Kanctl CLI installed (https://docs.kanister.io/tooling.html#kanctl)

To install kanister and related tools you can follow [this](https://docs.kanister.io/install.html#install) link.

**NOTE:**
The helm commands that are mentioned in this document are run with the helm version 3. If you are using other helm client version the commands may differ slightly.


### Install the same profile and blueprint on both cluster 

```
kubectl create -f copy-volume-data-bp.yaml

helm install profile kanister/profile \
    --namespace kanister \
    --version 0.61.0 \
    --set defaultProfile=true \
    --set location.type='s3Compliant' \
    --set location.bucket="<MY_BUCKET_NAME>" \
    --set location.region="<MY_AWS_REGION>" \
    --set aws.accessKey="<MY_AWS_ACCESS_KEY>" \
    --set aws.secretKey="<MY_AWS_SECRET_ACCESS_KEY>"
```

### Install kanctl 

Install [kanctl](https://github.com/kanisterio/kanister/releases/tag/0.61.0), it's a simple executable.
Add it to your path.


## Create a PVC in a namespace 

Switch your kubectl context to your source cluster.

Create the test-pvc ns and a PVC with data inside
```
kubectl create ns test-pvc
kubectl create -f my-pvc.yaml -n test-pvc
kubectl create -f create-data.yaml -n test-pvc
kubectl create -f explore-pvc.yaml -n test-pvc
```

Create data pod will just last the time to execute the command `echo 'hello world' >> /my-pvc/my-file`. This is for creating some test data and validating we successfully migrate it.

Use explore-data to explore the data in the PVC 
```
kubectl exec -it -n test-pvc explore-data -- /bin/sh
/ # ls /my-pvc/
my-file
/ # cat my-pvc/my-file 
hello world
/ # exit 
```

Delete the pod explore-data to free the PVC.
```
kubectl delete -f explore-pvc.yaml
```

## Run a backup of the PVC with kanctl 

### Create a backup 

Create an Actionset to trigger the backup action of the copy-volume blueprint.
```
kanctl create actionset -n kanister -a backup -v test-pvc/my-pvc -b copy-volume-bp -p default-profile
actionset backup-mvwhh created
```

Check the status of the actionset 
```
kubectl get actionset backup-mvwhh -n kanister -o json | jq '.status.state'
"complete"
```

PVC content has been sent to the S3 bucket defined in default-profile.

### Prepare a restore-action for the destination cluster.

Create a dry-run restore action that will execute in the destination cluster.

```
kanctl --namespace kanister create actionset -a restore --from backup-mvwhh --dry-run > restore-backup-mvwhh.yaml
```

## Restore in the destination cluster

Switch your kubectl context to the destination cluster

Create the namespace and the empty PVC
```
kubectl create ns test-pvc
kubectl create -f my-pvc.yaml -n test-pvc
```

Execute the restore from the previous backup.
```
kubectl create -f restore-backup-mvwhh.yaml -n kanister
```

Now explore the data 
```
kubectl create -f explore-pvc.yaml -n test-pvc
kubectl exec -it -n test-pvc explore-data -- /bin/sh
/ # ls /my-pvc/
my-file
/ # cat my-pvc/my-file 
hello world
/ # exit 
```

PVC is restored.
