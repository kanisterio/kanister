#!/usr/bin/env bash

### Created by cluster-etcd-operator. DO NOT edit.

set -o errexit
set -o pipefail
set -o errtrace

# example
# ./cluster-restore.sh $path-to-backup

if [[ $EUID -ne 0 ]]; then
  echo "This script must be run as root"
  exit 1
fi

source /etc/kubernetes/static-pod-resources/etcd-certs/configmaps/etcd-scripts/etcd.env
source /etc/kubernetes/static-pod-resources/etcd-certs/configmaps/etcd-scripts/etcd-common-tools

BACKUP_DIR="$1"
SNAPSHOT_FILE=$(ls -vd "${BACKUP_DIR}"/etcd-backup.db | tail -1) || true


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
