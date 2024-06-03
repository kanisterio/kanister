#!/usr/bin/env bash
# Copyright 2024 The Kanister Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


set -o errexit
set -o nounset
set -o xtrace
set -o pipefail

PACKAGE_FOLDER=${PACKAGE_FOLDER:?"PACKAGE_FOLDER not specified"}
HELM_RELEASE_REPO_URL=${HELM_RELEASE_REPO_URL:?"HELM_RELEASE_REPO_URL not specified"}
HELM_RELEASE_REPO_INDEX=${HELM_RELEASE_REPO_INDEX:?"HELM_RELEASE_REPO_INDEX not specified"}

main() {
    version=${1:?"chart version not specified"}

    ## Cleanup old package folder
    if [[ -d ${PACKAGE_FOLDER} ]]
    then
        rm -fr ${PACKAGE_FOLDER}
    fi
    mkdir ${PACKAGE_FOLDER}

    # Build kanister-operator chart archive
    helm package helm/kanister-operator --version ${version} -d ${PACKAGE_FOLDER}

    local repo_args="--url ${HELM_RELEASE_REPO_URL}"

    if curl ${HELM_RELEASE_REPO_INDEX} -o ${PACKAGE_FOLDER}/cur_index.yaml
    then
        repo_args="${repo_args} --merge ${PACKAGE_FOLDER}/cur_index.yaml"
    fi

    helm repo index ${PACKAGE_FOLDER} ${repo_args}
}

main $@
