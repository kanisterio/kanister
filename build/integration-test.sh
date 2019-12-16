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
INTEGRATION_TEST_DIR=pkg/testing
# Degree of parallelism for integration tests
DOP="8"
TEST_TIMEOUT="30m"

check_dependencies() {
    # Check if minio is already deployed
    if helm status minio -n minio > /dev/null 2>&1 ; then
        # Setting env vars to access MinIO
        export AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
        export AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
        export AWS_REGION="us-west-2"
        export LOCATION_ENDPOINT="http://minio.minio.svc.cluster.local:9000"
    else
        echo "Please install MinIO using 'make install-minio' and try again."
        exit 1
    fi
}

check_dependencies
echo "Running integration tests:"
pushd ${INTEGRATION_TEST_DIR}
TAGS="-tags=integration -timeout ${TEST_TIMEOUT} -check.suitep ${DOP}"
go test -v ${TAGS} -installsuffix "static" . -check.v
popd
