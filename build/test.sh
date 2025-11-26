#!/bin/bash

# Copyright 2019 The Kanister Authors.
#
# Copyright 2016 The Kubernetes Authors.
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

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

export CGO_ENABLED=0
export GO111MODULE=on
echo "Git commit present: ${COMMIT_SHA}"
echo "Git commit used: ${GIT_COMMIT}"
export GIT_COMMIT="${GIT_COMMIT}"

TEST_FILTER="${TEST_FILTER:-}"
GOCHECK_FILTER=""
if [ -n "${TEST_FILTER}" ]; then
    echo "Using test filter ${TEST_FILTER}"
    GOCHECK_FILTER="-check.f ${TEST_FILTER}"
fi

TARGETS=$(for d in "$@"; do echo ./$d/...; done)

check_dependencies() {
    # Check if minio is already deployed. We suppress only `stdout` and not `stderr` to make sure we catch errors if `helm status` fails
    if helm status minio -n minio 1> /dev/null ; then
        # Setting env vars to access MinIO
        export AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
        export AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
        export AWS_REGION="us-west-2"
        export LOCATION_ENDPOINT="http://localhost:9000"
        export LOCATION_CLUSTER_ENDPOINT="http://minio.minio.svc.cluster.local:9000"
        export TEST_REPOSITORY_ENCRYPTION_KEY="testKopiaRepoPassword"
        unset AWS_SESSION_TOKEN
        export USE_MINIO="true"
        export LOCAL_MINIO="${LOCAL_MINIO:-false}"
    else
        echo "Please install MinIO using 'make install-minio' and try again."
        exit 1
    fi

    # A test (CRDSuite) that runs as part of `make test` requires at least one CRD to
    # be present on the cluster. That's why we are checking that `csi-hostpath-driver`
    # installed before running tests.
    if ! ${SCRIPT_DIR}/local_kubernetes.sh check_csi_hostpath_driver_installed ; then
        echo "CRDs are not installed on the cluster but a test (CRDSuite) requires at least one CRD to be available on the cluster."\
        " One can be installed by running 'make install-csi-hostpath-driver' command."
        exit 1
    fi
}

check_dependencies

## make pod ready timeout shorter for tests
export TIMEOUT_WORKER_POD_READY="5"

## Split function and controller tests
base_packages=$(go list ${TARGETS} | grep -v pkg/function | grep -v pkg/controller | xargs echo);
controller_packages=$(go list ${TARGETS}  | grep pkg/controller | xargs echo);
functions_packages=$(go list ${TARGETS}  | grep pkg/function | xargs echo);

TEST_CONTROLLER="${TEST_CONTROLLER:-true}"
TEST_FUNCTIONS="${TEST_FUNCTIONS:-true}"
TEST_BASE="${TEST_BASE:-true}"

test_packages=""
if [ ${TEST_CONTROLLER} == "true" ]; then
    test_packages="${test_packages} ${controller_packages}"
fi
if [ ${TEST_FUNCTIONS} == "true" ]; then
    test_packages="${test_packages} ${functions_packages}"
fi
if [ ${TEST_BASE} == "true" ]; then
    test_packages="${test_packages} ${base_packages}"
fi

echo "Running tests:"
echo "Test packages: '${test_packages}'"

go test -v -timeout 30m -installsuffix "static" ${test_packages} -check.v ${GOCHECK_FILTER}
echo

echo "PASS"
echo
