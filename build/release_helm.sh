#!/usr/bin/env bash

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
    helm init --client-only
    helm dep update ${chart_path}
    local out=$(helm package ${chart_path} --version ${version} -d ${package_folder})
    [[ ${out} =~ ^.*/(.*\.tgz)$ ]]
    local chart_tar=${BASH_REMATCH[1]}
    local repo_args=""

    if ! [ -z ${HELM_RELEASE_REPO_URL} ]
    then
        repo_args="--url ${HELM_RELEASE_REPO_URL}"
    fi

    if aws s3 cp ${HELM_RELEASE_BUCKET}/index.yaml ${package_folder}/cur_index.yaml
    then
        repo_args="${repo_args} --merge ${package_folder}/cur_index.yaml"
    fi

    helm repo index ${package_folder} ${repo_args}

    if ! [ -z ${HELM_RELEASE_BUCKET} ]
    then
        echo "Uploading chart and index file"
        aws s3 cp ${package_folder}/${chart_tar} ${HELM_RELEASE_BUCKET}
        aws s3 cp ${package_folder}/index.yaml ${HELM_RELEASE_BUCKET}
    fi
}

main() {
    version=${1:?"chart version not specified"}
    # from examples folder
    local -a kanister_charts=( "kanister-mongodb-replicaset" "kanister-mysql" "kanister-postgresql")
    for chart_name in "${kanister_charts[@]}"
    do
        release_helm_charts examples/helm/kanister/${chart_name} "${version}"
    done

    # from helm folder
    release_helm_charts helm/profile "${version}"
}

main $@
