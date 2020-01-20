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

TMP_DIR=/tmp/minishift

install_minishift ()
{
    echo 'Started Deploying minishift...'
    echo
    echo 'Downloading minishift...'
    mkdir -p ${TMP_DIR}
    wget https://github.com/minishift/minishift/releases/download/v1.34.2/minishift-1.34.2-linux-amd64.tgz  -P ${TMP_DIR}
    tar zxvf ${TMP_DIR}/minishift-1.34.2-linux-amd64.tgz -C ${TMP_DIR}
    cp ${TMP_DIR}/minishift-1.34.2-linux-amd64/minishift ${GOPATH}/bin
    echo 'minishift was downloaded'
    echo 
    echo 'Starting minishift...'
    minishift start --vm-driver=${1}
    echo 'minishift was started successfully.' 
    echo
    echo 'Copying OpenShift client to correct location...'
    cp  /home/user/.minishift/cache/oc/v3.11.0/linux/oc ${GOPATH}/bin
    echo 'Success, you are ready to use minishift.'
}


uninstall_minishift ()
{
    minishift stop
    echo   
    minishift delete -f
    rm -rf ${GOPATH}/bin/minishift
    rm -rf ${GOPATH}/bin/oc
    rm -rf ${TMP_DIR}
    echo 'minishift was deleted successfully.'
}

[ ${#@} -gt 0 ] || usage
case "${1}" in
        # Alphabetically sorted
        install_minishift)
            time -p install_minishift "${2}"
            ;;
        uninstall_minishift)
            time -p uninstall_minishift
            ;;
        *)
            usage
            exit 1
esac