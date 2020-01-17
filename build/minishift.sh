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

install_minishift ()
{
    echo 'Started Deploying minishift...'
    echo 'Downloading minishift...'
    mkdir -p /tmp/minishift/bin
    wget https://github.com/minishift/minishift/releases/download/v1.34.2/minishift-1.34.2-linux-amd64.tgz  -P /tmp/minishift/
    tar zxvf /tmp/minishift/minishift-1.34.2-linux-amd64.tgz -C /tmp/minishift/bin
    #cp /tmp/minishift/minishift-1.34.2-linux-amd64/minishift /usr/local/bin/
    export PATH=/tmp/minishift/bin:$PATH
    echo 'minishift was downloaded'
    echo 
    echo 'Starting minishift...'
    minishift start --vm-driver=${1}
    echo 'minishift was started successfully.' 
    echo
    echo 'Setting path variable...'
    eval $(minishift oc-env)
    echo 'Success, you are ready to use minishift.'
}


uninstall_minishift ()
{
    echo 'Stopping minishift...'
    minishift stop
    echo
    echo 'Deleting minishift...'
    minishift delete -f
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