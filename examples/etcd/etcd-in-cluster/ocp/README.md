This document is going to show you how you can backup the ETCD of your OpenShift cluster. The
commands are run into an [OCP](https://www.openshift.com/products/container-platform) but it should work on any other OpenShift cluster.


## Prerequisites Details

* OpenShift (OCP) cluster
* PV support on the underlying infrastructure
* Kanister version 0.32.0 with `profiles.cr.kanister.io` CRD, [`kanctl`](https://docs.kanister.io/tooling.html#install-the-tools) Kanister tool installed


# Integrating with Kanister

When we say integrating with Kanister we mean creating some CRs, for example Blurprint and Actionset
that would help perform the actions on the the ETCD instance that we are running.

## Create Profile resource

```bash
» kanctl create profile s3compliant --access-key <aws-access-key> \
        --secret-key <aws-secret-key> \
        --bucket <bucket-name> --region <region-name> \
        --namespace kanister
secret 's3-secret-7umv91' created
profile 's3-profile-nnvmm' created
```
This command creates a profile which we will use later.

The command will configure a location where artifacts resulting from Kanister
data operations such as backup should go. This is stored as a `profiles.cr.kanister.io`
*CustomResource (CR)* which is then referenced in Kanister ActionSets. Every ActionSet
requires a Profile reference to complete the action. This CR (`profiles.cr.kanister.io`)
can be shared between Kanister-enabled application instances.


## Create Blueprint

Before actually creating the BluePrint, we will have to create a secret in the same namespace where your ETCD pod is running. This
secret is going to have the name of the format `etcd-<etcd-pod-namespace>` with these fields

- **endpoints** : ETCD server client listen URL, https://[127.0.0.1]:2379
- **labels** : These labels will be used to identify the etcd pods that are running, for ex `app=etcd,etcd=true`

Below command can be used to create the secret, assuming the etcd pods are running in the `openshift-etcd` namespace

```
» oc create secret generic etcd-openshift-etcd \
    --from-literal=endpoints=https://10.0.133.5:2379 \
    --from-literal=labels=app=etcd,etcd=true \
    --namespace openshift-etcd
secret/etcd-openshift-etcd created
```

Once the secret is created below command can be used to create the BluePrint

```
» oc apply -f etcd-incluster-ocp-blueprint.yaml -n kanister
blueprint.cr.kanister.io/etcd-blueprint configured
```

## Protect the Application

We can now take snapshot of the ETCD server that is running by creating backup actionset that is going to execute backup phase from the blueprint that we have
created above

**Note**

Pleae make sure to change the **profile-name**,  **namespace-name** and **blueprint name** in the `backup-actionset.yaml` manifest file. Where `namespace-name` is the namespace where the ETCD pods are running.

```
# find the profile name
» oc get profiles.cr.kanister.io -n kanister
NAME               AGE
s3-profile-2lhk8   52s

# fine the BluePrint name
» oc get blueprint -n kanister
NAME             AGE
etcd-blueprint   85m

# create actionset
» oc create -f backup-actionset.yaml --namespace kanister
actionset.cr.kanister.io/backup-4f6jn created

# you can check the status of the actionset to make sure it has been completed
» oc describe actionset -n kanister backup-4f6jn
Name:         backup-4f6jn
Namespace:    kanister
Labels:       <none>
...
...
Events:
  Type    Reason           Age   From                 Message
  ----    ------           ----  ----                 -------
  Normal  Started Action   3m    Kanister Controller  Executing action backup
  Normal  Started Phase    3m    Kanister Controller  Executing phase takeSnapshot
  Normal  Ended Phase      3m    Kanister Controller  Completed phase takeSnapshot
  Normal  Started Phase    3m    Kanister Controller  Executing phase uploadSnapshot
  Normal  Ended Phase      2m    Kanister Controller  Completed phase uploadSnapshot
  Normal  Started Phase    2m    Kanister Controller  Executing phase removeSnapshot
  Normal  Ended Phase      2m    Kanister Controller  Completed phase removeSnapshot
  Normal  Update Complete  2m    Kanister Controller  Updated ActionSet 'backup-4f6jn' Status->complete
```

## Restore ETCD cluster

To restore the ETCD cluster we can follow the [documentation](https://docs.openshift.com/container-platform/4.5/backup_and_restore/disaster_recovery/scenario-2-restoring-cluster-state.html) that is provided the OpenShift team.
But we will have to make some modification into the restore script (`cluster-restore.sh`) because default
restore script expects the static pods manifest as well and in our case we didnt backup the satic pod manifests.

Or in other words you have to follow all the steps that are mentioned in the above documentation but instead of `cluster-restore.sh` script below steps should be followed

```

source /etc/kubernetes/static-pod-resources/etcd-certs/configmaps/etcd-scripts/etcd.env
source /etc/kubernetes/static-pod-resources/etcd-certs/configmaps/etcd-scripts/etcd-common-tools

SNAPSHOT_FILE=$(ls -vd "${BACKUP_DIR}"/snapshot*.db | tail -1) || true


if [ ! -f "${SNAPSHOT_FILE}" ]; then
  echo "etcd snapshot ${SNAPSHOT_FILE} does not exist"
  exit 1
fi

# Move manifests and stop static pods
#ASSET_DIR="/home/core/assets"
#MANIFEST_STOPPED_DIR="${ASSET_DIR}/manifests-stopped"
if [ ! -d "$MANIFEST_STOPPED_DIR" ]; then
  mkdir -p $MANIFEST_STOPPED_DIR
fi



#ETCD_DATA_DIR_BACKUP="/var/lib/etcd-backup"
if [ ! -d ${ETCD_DATA_DIR_BACKUP} ]; then
  mkdir -p ${ETCD_DATA_DIR_BACKUP}
fi

# backup old data-dir
#ETCD_DATA_DIR="/var/lib/etcd"
if [ -d "${ETCD_DATA_DIR}/member" ]; then
  if [ -d "${ETCD_DATA_DIR_BACKUP}/member" ]; then
    echo "removing previous backup ${ETCD_DATA_DIR_BACKUP}/member"
    rm -rf ${ETCD_DATA_DIR_BACKUP}/member
  fi
  echo "Moving etcd data-dir ${ETCD_DATA_DIR}/member to ${ETCD_DATA_DIR_BACKUP}"
  mv ${ETCD_DATA_DIR}/member ${ETCD_DATA_DIR_BACKUP}/
fi

# Restore static pod resources
#CONFIG_FILE_DIR="/etc/kubernetes"


# Copy snapshot to backupdir
cp -p ${SNAPSHOT_FILE} ${ETCD_DATA_DIR_BACKUP}/snapshot.db

echo "starting restore-1etcd static pod"
#RESTORE_ETCD_POD_YAML="${CONFIG_FILE_DIR}/static-pod-resources/etcd-certs/configmaps/restore-etcd-pod/pod.yaml"
cp -p ${RESTORE_ETCD_POD_YAML} ${MANIFEST_DIR}/etcd-pod.yaml



```
