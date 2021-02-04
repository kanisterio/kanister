#!/usr/bin/env bash
# Copyright 2019 The Kanister Authors.
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

export HELM_RELEASE_BUCKET=s3://charts.kanister.io
export HELM_RELEASE_REPO_URL=https://charts.kanister.io
readonly TMP_DIR=$(mktemp -d /tmp/kanister_build.XXXX);

release_helm_charts() {
    local chart_path=${1:?"Helm chart is not specified"}
    local version=${2:?"chart version not specified"}
    local package_folder=${TMP_DIR}/helm_package

    if [[ -d ${package_folder} ]]
    then
        rm -fr ${package_folder}
    fi

    mkdir ${package_folder}
    local out=$(helm package ${chart_path} --version ${version} -d ${package_folder})
    [[ ${out} =~ ^.*/(.*\.tgz)$ ]]
    local chart_tar=${BASH_REMATCH[1]}
    local repo_args="--url ${HELM_RELEASE_REPO_URL}"

    if aws s3 cp ${HELM_RELEASE_BUCKET}/index.yaml ${package_folder}/cur_index.yaml
    then
        repo_args="${repo_args} --merge ${package_folder}/cur_index.yaml"
    fi

    helm repo index ${package_folder} ${repo_args}

    echo "Uploading chart and index file"
    aws s3 cp ${package_folder}/${chart_tar} ${HELM_RELEASE_BUCKET}
    aws s3 cp ${package_folder}/index.yaml ${HELM_RELEASE_BUCKET}
    cp -f ${package_folder}/index.yaml ${HELM_HOME:-${HOME}/.helm}/repository/cache/kanister-index.yaml
    cp -f ${package_folder}/${chart_tar} ${HELM_HOME:-${HOME}/.helm}/cache/archive/
}

main() {
    version=${1:?"chart version not specified"}

    helm init --client-only --stable-repo-url https://charts.helm.sh/stable

    helm repo add kanister ${HELM_RELEASE_REPO_URL}
    release_helm_charts helm/profile "${version}"

    # Release kanister-operator chart
    release_helm_charts helm/kanister-operator "${version}"

}

main $@
