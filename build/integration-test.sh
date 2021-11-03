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
DOP="4"
TEST_TIMEOUT="30m"
# Set default options
TEST_OPTIONS="-tags=integration -timeout ${TEST_TIMEOUT} -check.suitep ${DOP}"
# Regex to match apps to run in short mode
SHORT_APPS="^PostgreSQL$|^PITRPostgreSQL|^MySQL$|Elasticsearch|^MongoDB$|Maria"
# OCAPPS has all the apps that are to be tested against openshift cluster
OC_APPS3_11="MysqlDBDepConfig$|MongoDBDepConfig$|PostgreSQLDepConfig$"
OC_APPS4_4="MysqlDBDepConfig4_4|MongoDBDepConfig4_4|PostgreSQLDepConfig4_4"
OC_APPS4_5="MysqlDBDepConfig4_5|MongoDBDepConfig4_5|PostgreSQLDepConfig4_5"
OC_APPS4_7="MysqlDBDepConfig4_7|MongoDBDepConfig4_7|PostgreSQLDepConfig4_7"

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

usage() {
    cat <<EOM
Usage: ${0} <app-type>
Where app-type is one of [short|all]:
  short: Runs e2e integration tests for part of apps
  all: Runs e2e integration tests for all apps
  OR
  You can also provide regex to match apps you want to run.
  openshift ocp_version=<ocp_version>: Runs e2e integration tests for specific version of OpenShift apps, OCP version can be provided using ocp_version argument. Currently supported versions are 3.11, 4.4 and 4.5.

EOM
    exit 1
}

[ ${#@} -gt 0 ] || usage
case "${1}" in
    all)
        TEST_APPS=".*"
        ;;
    short)
        # Run only part of apps
        TEST_APPS=${SHORT_APPS}
        ;;
    openshift)
        # TODO:// make sure the argument is named ocp_version
        if [[ ${#@} == 1 ]]; then
            usage
        fi

        case "${2}" in
            "3.11")
                TEST_APPS=${OC_APPS3_11}
                ;;
            "4.4")
                TEST_APPS=${OC_APPS4_4}
                ;;
            "4.5")
                TEST_APPS=${OC_APPS4_5}
                ;;
            "4.7")
                TEST_APPS=${OC_APPS4_7}
                ;;
            *)
                usage
                ;;
        esac
        ;;
    *)
        TEST_APPS=${1}
        ;;
esac

check_dependencies
echo "Running integration tests:"
pushd ${INTEGRATION_TEST_DIR}
go test -v ${TEST_OPTIONS} -check.f "${TEST_APPS}" -installsuffix "static" . -check.v
popd
