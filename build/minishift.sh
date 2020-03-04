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

export TMP_DIR=/tmp/minishift
export MINISHIFT_VERSION=1.34.2
export OC_VERSION=3.11.0

start_minishift ()
{
    echo 'Started Deploying minishift...'
    echo
    echo 'Downloading minishift...'
    mkdir -p ${TMP_DIR}
    wget https://github.com/minishift/minishift/releases/download/v${MINISHIFT_VERSION}/minishift-${MINISHIFT_VERSION}-linux-amd64.tgz  -P ${TMP_DIR}
    tar zxvf ${TMP_DIR}/minishift-${MINISHIFT_VERSION}-linux-amd64.tgz -C ${TMP_DIR}
    cp ${TMP_DIR}/minishift-${MINISHIFT_VERSION}-linux-amd64/minishift ${GOPATH}/bin
    echo 'minishift was downloaded'
    echo
    echo 'Starting minishift...'
    minishift start --vm-driver=${1}
    echo 'minishift was started successfully.'
    echo
    echo 'Copying OpenShift client to correct location...'
    cp  ${HOME}/.minishift/cache/oc/v${OC_VERSION}/linux/oc ${GOPATH}/bin
    oc login -u system:admin
    # https://github.com/minio/minio/issues/6524#issuecomment-451689375
    oc adm policy add-scc-to-group anyuid system:authenticated
    echo 'Success, you are ready to use minishift.'
}


stop_minishift ()
{
    minishift stop
    echo
    minishift delete -f
    rm -rf ${GOPATH}/bin/minishift
    rm -rf ${GOPATH}/bin/oc
    rm -rf ${TMP_DIR}
    echo 'minishift was deleted successfully.'
}


usage() {
    cat <<EOM
Usage: ${0} <operation>
Where operation is one of the following:
  start_minishift vm-driver=kvm: installs minishift
  stop_minishift : uninstalls minishift
EOM
    exit 1
}

[ ${#@} -gt 0 ] || usage
case "${1}" in
        # Alphabetically sorted
        start_minishift)
            # check if vm-driver was provided or not
            if [ $# -ne 2 ]
            then
                echo 'Please provide vm driver using vm-driver flag.'
                exit 1
            fi
            time -p start_minishift "${2}"
            ;;
        stop_minishift)
            time -p stop_minishift
            ;;
        *)
            usage
            exit 1
esac
