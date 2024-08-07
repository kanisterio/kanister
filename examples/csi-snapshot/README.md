# Backup and restore using CSI VolumeSnapshot

## Introduction
VolumeSnapshots provide Kubernetes users with a standardized way to copy a volume's contents at a particular point in time without creating an entirely new volume. This functionality enables, for example, database administrators to backup databases before performing edit or delete modifications.
This example demonstrates Kanister's ability to protect an application called Time-Logger using CSI VolumeSnapshots. Time-Logger application is contrived but useful for demonstrating Kanister's features. The application appends the current time to a log file every second.

## Prerequisites

- Helm 3 installed
- Kubernetes 1.16+ with Beta APIs enabled
- Kanister controller version 0.110.0 installed in the cluster, let's assume in namespace `kanister`
- Kanctl CLI installed (https://docs.kanister.io/tooling.html#install-the-tools)
- VolumeSnapshot CRDs, Snapshot Controller & a CSI Driver

## Install application

Install Time-Logger application.

```bash
# Create namespace
$ kubectl create namespace time-log

# Create the Deployment and PersistentVolumeClaim in time-log namespace
$ kubectl -n time-log create -f ./examples/time-log/time-logger-deployment.yaml

# Run a shell inside time-logger deployment's pod
$ kubectl exec -it $(kubectl get pods -n time-log -l app=time-logger -o=jsonpath='{.items[0].metadata.name}') -n time-log -- /bin/bash

# Make note of the first entry recorded in the /var/log/time.log
$ head --lines=1 /var/log/time.log
Sun Jan 23 08:54:39 UTC 2022
```

## Protect the Application

### Create Blueprint

Create the Blueprint in `kanister` namespace

> **Note**:  This example uses a Kubernetes cluster on DigitalOcean. Therefore the `snapshotClass` and `storageClass` in the following `./examples/csi-snapshot/csi-snapshot-blueprint.yaml` file are set to `do-block-storage`. Change the arguments appropriately before creating the blueprint.

```bash
$ kubectl create -f ./examples/csi-snapshot/csi-snapshot-blueprint.yaml -n kanister
```

### Backup the application data

Create a snapshot of application data using `backup` action defined in blueprint. One of the easiest ways to do so is by using `kanctl` utility.

```bash
# Create Actionset
# Make sure the value of blueprint matches the name of blueprint created earlier

$ kanctl create actionset --action backup --namespace kanister --blueprint csi-snapshot-bp --deployment time-log/time-logger
actionset backup-mlvcv created

$ kubectl --namespace kanister get actionset
NAME                         AGE
backup-mlvcv                 112s

# View the status of the actionset
# Make sure the name of the actionset here matches the name of the actionset created above
$ kubectl --namespace kanister describe actionset backup-mlvcv

# Check the CSI VolumeSnapshot created
$ kubectl -n time-log get volumesnapshot
NAME                          READYTOUSE   SOURCEPVC      SOURCESNAPSHOTCONTENT   RESTORESIZE   SNAPSHOTCLASS            SNAPSHOTCONTENT                               CREATIONTIME   AGE
time-log-pvc-snapshot-nxj4g   true         time-log-pvc                           1Gi           do-block-storage   snapcontent-13bc2d1c-6717-47a2-a0b7-4a2f76bd2cb4   58s            58s
```

## Disaster strikes!

Let's say someone accidentally deleted the log file at path `/var/log`

```bash
# Run a shell inside time-logger deployment's pod
$ kubectl exec -it $(kubectl get pods -n time-log -l app=time-logger -o=jsonpath='{.items[0].metadata.name}') -n time-log -- /bin/bash

# Remove the log file
$ rm /var/log/time.log

# Check the first entry recorded in the /var/log/time.log again. It is now replaced with a new entry.
# This is because Time-Logger app recreates the log file once it's deleted and starts adding newer entries to it.
$ head --lines=1 /var/log/time.log
Sun Jan 23 09:15:43 UTC 2022
```

## Restore application

Use the backup created earlier to restore the application data. This can be achieved using `kanctl` again by creating the restore action.

```bash
# Make sure to use correct backup actionset name here
$ kanctl --namespace kanister create actionset --action restore --from backup-mlvcv
actionset restore-backup-mlvcv-6z9xn created

# Use the following command to check events in the actionset
$ kubectl --namespace kanister describe actionset restore-backup-mlvcv-6z9xn
```

## Verify the restored application data

To verify restore action, check the first entry of the file `/var/log/time.log` and confirm that it's been restored with the original value.

```bash
# Run a shell inside time-logger deployment's pod
$ kubectl exec -it $(kubectl get pods -n time-log -l app=time-logger -o=jsonpath='{.items[0].metadata.name}') -n time-log -- /bin/bash

$ head --lines=1 /var/log/time.log
Sun Jan 23 08:54:39 UTC 2022
```

## Delete the Artifacts

The CSI VolumeSnapshot created by the backup action can be cleaned up using the following command.

```bash
# Make sure to use correct backup actionset name here
$ kanctl --namespace kanister create actionset --action delete --from backup-mlvcv
actionset delete-backup-mlvcv-cq6bw created

# View the status of the ActionSet
$ kubectl --namespace kanister describe actionset delete-backup-glptq-cq6bw
```

## Cleanup

### Uninstalling the application

To uninstall the application delete `time-log` namespace.

```bash
# Remove time-log namespace
$ kubectl delete namespace time-log
```

### Delete CRs

Remove the blueprint.

```bash
$ kubectl delete blueprints.cr.kanister.io csi-snapshot-bp -n kanister
```

Remove the actionsets.

```bash
$ kubectl delete actionsets -n kanister --all
```
