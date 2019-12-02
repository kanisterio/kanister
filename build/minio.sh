#!/bin/bash

# Copyright 2019 The Kanister Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset

accessID="minio2a5b4f9214e3"
secretKey="miniob08c7b34c0b4603d"
s3Bucket="test.kanister.io"

installminio ()
{
    echo "Deploying minio..."
    # Add stable helm repo
    cmd="helm repo add stable https://kubernetes-charts.storage.googleapis.com &&\
         helm repo update"
    result=$(eval ${cmd})
    ret=$?
    if [[ $ret -ne "0" ]]; then
        echo "Failed to add helm repo"
        exit 1
    fi
    # helm install minio
    cmd="kubectl create ns minio && \
         helm install minio --namespace minio \
         --set accessKey=${accessID},secretKey=${secretKey} \
         --set defaultBucket.enabled=true,defaultBucket.name=${s3Bucket} \
         --set environment.MINIO_SSE_AUTO_ENCRYPTION=on \
         --set environment.MINIO_SSE_MASTER_KEY=my-minio-key:feb5bb6c5cf851e21dbc0376ca81012a9edc4ca0ceeb9df5064ccba2991ae9de \
         stable/minio --wait --timeout 3m"
    result=$(eval ${cmd})
    ret=$?
    if [[ $ret -ne "0" ]]; then
        echo "Failed to install minio"
        exit 1
    fi

    # export the creds
    export AWS_ACCESS_KEY_ID=${accessID}
    export AWS_SECRET_ACCESS_KEY=${secretKey}
    export AWS_REGION="us-east-1"
    export LOCATION_ENDPOINT="http://minio.minio.svc.cluster.local:9000"
    # skip rds-postgres app
    export SKIP_RDS_POSTGRES=true
    echo "minio deployment successful!"
}

uninstallminio ()
{
    echo "Removing minio..."
    cmd="helm delete minio --namespace minio && kubectl delete ns minio"
    result=$(eval ${cmd})
    ret=$?
    if [[ $ret -ne "0" ]]; then
        echo "Failed to remove minio"
        exit 1
    fi
    echo "minio removed successfully!"
}
