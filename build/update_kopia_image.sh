#!/bin/bash

# Copyright 2021 The Kanister Authors.
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

IMAGE_REGISTRY="ghcr.io/kanisterio"

set -o errexit
set -o nounset
set -o xtrace

readonly COMMIT_ID=${1:?"Commit id to build kopia image not specified"}
readonly KOPIA_REPO_ORG=${2-:"kopia"}
readonly IMAGE_TYPE=alpine
readonly TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
readonly IMAGE_BUILD_VERSION="${COMMIT_ID}"
readonly GH_PACKAGE_TARGET="ghcr.io/kanisterio/kopia"
readonly TAG="${IMAGE_TYPE}-${IMAGE_BUILD_VERSION}"


docker build \
    --tag "${GH_PACKAGE_TARGET}:${TAG}" \
    --build-arg "kopiaBuildCommit=${COMMIT_ID}" \
    --build-arg "timestamp=$(date +%Y%m%d-%H%M%S)" \
    --build-arg "kopiaRepoOrg=${KOPIA_REPO_ORG}" \
    --file ./docker/kopia-build/Dockerfile .

docker push $IMAGE_REGISTRY/kopia:$TAG
