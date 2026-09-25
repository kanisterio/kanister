#!/bin/bash

# Copyright 2026 The Kanister Authors.
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

# Adobe S3Mock is a lightweight, single-container S3 API mock built for
# test/CI use. https://github.com/adobe/S3Mock
# We apply it as a plain Deployment/Service (no helm chart) to keep the CI
# footprint minimal: one image pull, no chart fetch, no persistent volume.
S3MOCK_IMAGE="adobe/s3mock:5.2.3"
ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

install_s3mock ()
{
    echo "Deploying Adobe S3Mock..."

    # create s3mock namespace
    kubectl create ns s3mock

    # deploy a single-node S3 mock with the test bucket created at startup.
    # S3Mock accepts any access key/secret without validating them.
    kubectl apply -n s3mock -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: s3mock
  labels:
    app: s3mock
    release: s3mock
spec:
  replicas: 1
  selector:
    matchLabels:
      app: s3mock
      release: s3mock
  template:
    metadata:
      labels:
        app: s3mock
        release: s3mock
    spec:
      containers:
        - name: s3mock
          image: ${S3MOCK_IMAGE}
          ports:
            - containerPort: 9090
          env:
            - name: COM_ADOBE_TESTING_S3MOCK_STORE_INITIAL_BUCKETS
              value: "${S3_BUCKET}"
          readinessProbe:
            tcpSocket:
              port: 9090
            initialDelaySeconds: 2
            periodSeconds: 2
          resources:
            requests:
              memory: 512Mi
---
apiVersion: v1
kind: Service
metadata:
  name: s3mock
spec:
  selector:
    app: s3mock
    release: s3mock
  ports:
    - port: 9000
      targetPort: 9090
EOF

    kubectl -n s3mock rollout status deployment/s3mock --timeout=3m

    # export default creds for s3mock
    echo
    echo "Use following creds to access S3Mock"
    echo AWS_ACCESS_KEY_ID="${ACCESS_KEY_ID}"
    echo AWS_SECRET_ACCESS_KEY="${SECRET_ACCESS_KEY}"
    echo AWS_REGION="us-west-2"
    echo LOCATION_ENDPOINT="http://s3mock.s3mock.svc.cluster.local:9000"
    echo
    echo "s3mock deployment successful!"
}

uninstall_s3mock ()
{
    echo "Removing s3mock..."
    kubectl delete ns s3mock
}

usage() {
    cat <<EOM
Usage: ${0} <operation>
Where operation is one of the following:
  install_s3mock: installs s3mock on k8s cluster
  uninstall_s3mock: uninstalls s3mock
EOM
    exit 1
}

[ ${#@} -gt 0 ] || usage
case "${1}" in
        # Alphabetically sorted
        install_s3mock)
            time -p install_s3mock
            ;;
        uninstall_s3mock)
            time -p uninstall_s3mock
            ;;
        *)
            usage
            exit 1
esac
