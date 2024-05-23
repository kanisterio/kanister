#!/bin/sh

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
set -o xtrace

build_licenses_info_image() {
    local src_dir="$(pwd)"
    local target_file="$(pwd)/licenses"
    local mount_cmd="-v $(pwd):$(pwd)"
    # Checking if executed inside docker container
    if [ -n "${CONTAINER_NAME:-}" ]; then
        mount_cmd="--volumes-from ${CONTAINER_NAME}"
    elif grep docker /proc/1/cgroup -qa; then
        mount_cmd="--volumes-from $(grep docker -m 1 /proc/self/cgroup|cut -d/ -f3)"
    fi
    if [ -z "${ARCH:-""}" ]; then
        echo "ARCH must be set"
        exit 1
    fi
    docker run --rm ${mount_cmd} \
        "ghcr.io/kanisterio/license-extractor:4e0a91a" \
        --mode merge \
        --source ${src_dir} \
        --target ${target_file}\
        --overwrite > /dev/null
}

if [ -z "${BIN:-""}" ]; then
    echo "BIN must be set"
    exit 1
fi
if [ -z "${ARCH:-""}" ]; then
    echo "ARCH must be set"
    exit 1
fi
if [ -z "${IMAGE:-""}" ]; then
    echo "IMAGE must be set"
    exit 1
fi
if [ -z "${VERSION:-""}" ]; then
    echo "VERSION must be set"
    exit 1
fi
if [ -z "${SOURCE_BIN:-""}" ]; then
    echo "SOURCE_BIN must be set"
    exit 1
fi

build_licenses_info_image

sed                                \
    -e "s|ARG_BIN|${BIN}|g"        \
    -e "s|ARG_ARCH|${ARCH}|g"      \
    -e "s|ARG_SOURCE_BIN|${SOURCE_BIN}|g" \
    Dockerfile.in > .dockerfile-${ARCH}
docker buildx build --push --pull --sbom=${GENERATE_SBOM:-false} ${baseimagearg:-} --build-arg kanister_version=${VERSION} -t ${IMAGE}:${VERSION} --platform linux/${ARCH}  -f .dockerfile-${ARCH} .
docker images -q ${IMAGE}:${VERSION}
