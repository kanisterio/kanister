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

TEST_FILTER="${TEST_FILTER:-}"
GOCHECK_FILTER=""
if [ -n "${TEST_FILTER}" ]; then
    echo "Using test filter ${TEST_FILTER}"
    GOCHECK_FILTER="-check.f ${TEST_FILTER}"
fi

TARGETS=$(for d in "$@"; do echo ./$d/...; done)

echo -n "Checking gofmt: "
ERRS=$(find "$@" -type f -name \*.go | xargs gofmt -l 2>&1 || true)
if [ -n "${ERRS}" ]; then
    echo "FAIL - the following files need to be gofmt'ed:"
    for e in ${ERRS}; do
        echo "    $e"
    done
    echo
    exit 1
fi
echo "PASS"
echo

echo -n "Checking go vet: "
ERRS=$(go vet ${TARGETS} 2>&1 || true)
if [ -n "${ERRS}" ]; then
    echo "FAIL"
    echo "${ERRS}"
    echo
    # TODO: Renable govet. Currently generated code fails to pass go vet. report,
    # but don't exit on failures.
    #exit 1
fi
echo

check_dependencies() {
    # Check if minio is already deployed. We suppress only `stdout` and not `stderr` to make sure we catch errors if `helm status` fails
    if helm status minio -n minio 1> /dev/null ; then
        # Setting env vars to access MinIO
        export S3_COMPLIANT_AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
        export S3_COMPLIANT_AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
        export S3_COMPLIANT_AWS_REGION="us-west-2"
        export S3_COMPLIANT_LOCATION_ENDPOINT="http://minio.minio.svc.cluster.local:9000"
        export TEST_REPOSITORY_ENCRYPTION_KEY="testKopiaRepoPassword"
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

echo "Running tests:"
go test -v ${TARGETS} -list .
go test -v -installsuffix "static" ${TARGETS} -check.v ${GOCHECK_FILTER}
echo

echo "PASS"
echo
