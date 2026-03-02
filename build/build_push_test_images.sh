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
set -o pipefail

########################################
# Environment (override externally if needed)
########################################

ORG=${LOCAL_IMAGE_ORG:-"kanisterio"}
REGISTRY_ADDR=${LOCAL_IMAGE_REGISTRY:-"localhost:5000"}
REPOSITORY=${LOCAL_IMAGE_REPOSITORY:-"test_tools_image"}
IMAGE_TAG=${IMAGE_TAG:-"dev"}
PR_NUMBER=${PR_NUMBER:-"001"}

########################################
# Image matrix
########################################

IMAGES_AND_DOCKERFILES="
kanisterio/mongodb|docker/mongodb/Dockerfile
kanisterio/postgresql|docker/postgresql/Dockerfile
kanisterio/mssql-tools|docker/mssql-tools/Dockerfile
kanisterio/mysql-sidecar|docker/kanister-mysql/image/Dockerfile
kanisterio/postgres-kanister-tools|docker/postgres-kanister-tools/Dockerfile
"

########################################
# Build helpers
########################################

build_image() {
  local app_name="$1"
  local dockerfile="$2"

  local tag="pr-${PR_NUMBER}-${app_name}"
  local target_image="${REGISTRY_ADDR}/${ORG}/${REPOSITORY}:${tag}"

  echo "→ Building and pushing ${target_image}"

  docker buildx build \
    --platform=linux/amd64 \
    --push \
    --no-cache \
    -f "$dockerfile" \
    --build-arg "TOOLS_IMAGE=${TOOLS_IMAGE:-}" \
    -t "${target_image}" \
    . 2>&1 | sed "s|^|[${app_name}] |"
}

########################################
# Main
########################################

main() {
  echo "Using registry: ${REGISTRY_ADDR}"
  echo "Registry Org: " ${ORG}
  echo "Repository: ${REPOSITORY}"
  echo "PR Number: ${PR_NUMBER}"

  ########################################
  # 1️⃣ Build tools image first
  ########################################

  local tools_app="kanister-tools"
  local tools_dockerfile="docker/tools/Dockerfile"
  local tools_tag="pr-${PR_NUMBER}-${tools_app}"

  TOOLS_IMAGE="${REGISTRY_ADDR}/${ORG}/${REPOSITORY}:${tools_tag}"

  echo "→ Building tools image: ${TOOLS_IMAGE}"

  docker buildx build \
    --platform=linux/amd64 \
    --push \
    --no-cache \
    -f "${tools_dockerfile}" \
    -t "${TOOLS_IMAGE}" \
    . 2>&1 | sed "s|^|[kanister-tools] |"

  ########################################
  # 2️⃣ Build remaining images
  ########################################

  echo "Building remaining images..."

  while IFS='|' read -r image dockerfile; do
    [[ -n "${image}" ]] || continue

    app_name="${image##*/}"
    build_image "${app_name}" "${dockerfile}"

  done <<< "${IMAGES_AND_DOCKERFILES}"

  echo "All images pushed successfully."
}

main "$@"
