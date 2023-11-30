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

# Default bucket name
S3_BUCKET="tests.kanister.io"
MINIO_CHART_VERSION="5.0.14"

install_minio ()
{
    echo "Deploying minio..."
    # Add minio helm repo
    helm repo add minio https://charts.min.io/
    helm repo update

    # create minio namespace
    kubectl create ns minio

    # deploy minio
    helm install minio --version ${MINIO_CHART_VERSION} --namespace minio \
    --set resources.requests.memory=512Mi --set replicas=1 \
    --set persistence.enabled=false --set mode=standalone \
    --set buckets[0].name=${S3_BUCKET} \
    --set rootUser=AKIAIOSFODNN7EXAMPLE,rootPassword=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
    minio/minio --wait --timeout 3m

    # export default creds for minio
    # https://github.com/helm/charts/tree/master/stable/minio
    echo
    echo "Use following creds to access MinIO"
    echo AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
    echo AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
    echo AWS_REGION="us-west-2"
    echo LOCATION_ENDPOINT="http://minio.minio.svc.cluster.local:9000"
    echo
    echo "minio deployment successful!"
}

uninstall_minio ()
{
    echo "Removing minio..."
    helm delete minio -n minio
    kubectl delete ns minio
}

usage() {
    cat <<EOM
Usage: ${0} <operation>
Where operation is one of the following:
  install_minio: installs minio on k8s cluster
  uninstall_minio: uninstalls minio
EOM
    exit 1
}

[ ${#@} -gt 0 ] || usage
case "${1}" in
        # Alphabetically sorted
        install_minio)
            time -p install_minio
            ;;
        uninstall_minio)
            time -p uninstall_minio
            ;;
        *)
            usage
            exit 1
esac
