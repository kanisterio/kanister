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

REGISTRY_ADDR="localhost:5000"

IMAGES_AND_DOCKERFILES="
kanisterio/mongodb|docker/mongodb/Dockerfile
kanisterio/cassandra|docker/cassandra/Dockerfile
kanisterio/postgresql|docker/postgresql/Dockerfile
kanisterio/mssql-tools|docker/mssql-tools/Dockerfile
kanisterio/mysql-sidecar|docker/kanister-mysql/image/Dockerfile
kanisterio/es-sidecar|docker/kanister-elasticsearch/image/Dockerfile
kanisterio/postgres-kanister-tools|docker/postgres-kanister-tools/Dockerfile
kanisterio/kafka-adobe-s3-sink-connector|docker/kafka-adobes3Connector/image/adobeSink.Dockerfile
kanisterio/kafka-adobe-s3-source-connector|docker/kafka-adobes3Connector/image/adobeSource.Dockerfile
"

GITHUB_REPOSITORY_OWNER="kanisterio"
REPOSITORY="test_app_images"
IMAGE_TAG="${IMAGE_TAG:-dev}"

build_image() {
  local app_name="$1"
  local dockerfile="$2"

  local pr_number="${PR_NUMBER:-001}"
  local tag="pr-${pr_number}-${app_name}"
  local target_image="${REGISTRY_ADDR}/${REPOSITORY}:${tag}"

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

main() {
  echo "Determining environment..."

  if [[ "$MODE" == "ci" ]]; then
    if [[ -z "${PR_NUMBER:-}" ]]; then
      echo "❌ PR_NUMBER env variable not set"
      exit 1
    fi

    ORG="${GITHUB_REPOSITORY_OWNER}"
    REGISTRY_ADDR="ghcr.io/${ORG}"
  else
    REGISTRY_ADDR="localhost:5000"

    if ! curl -sf "http://${REGISTRY_ADDR}/v2/" >/dev/null; then
      echo "❌ Local registry not reachable at ${REGISTRY_ADDR}"
      exit 1
    fi
  fi

  echo "Registry: ${REGISTRY_ADDR}"

  ########################################
  # 1️⃣ Build tools image first
  ########################################

  TOOLS_APP="kanister-tools"
  TOOLS_DOCKERFILE="docker/tools/Dockerfile"

  pr_number="${PR_NUMBER:-001}"
  tools_tag="pr-${pr_number}-${TOOLS_APP}"
  TOOLS_IMAGE="${REGISTRY_ADDR}/${REPOSITORY}:${tools_tag}"

  echo "→ Building tools image first: ${TOOLS_IMAGE}"

 docker buildx build \
  --platform=linux/amd64 \
  --push \
  --no-cache \
  -f "${TOOLS_DOCKERFILE}" \
  -t "${TOOLS_IMAGE}" \
  . 2>&1 | sed "s|^|[kanister-tools] |"

  ########################################
  # 2️⃣ Build remaining images
  ########################################

  echo "Building remaining images..."

  while IFS='|' read -r image dockerfile; do
    [[ -n "$image" ]] || continue

    app_name="${image##*/}"

    build_image "$app_name" "$dockerfile"

  done <<< "$IMAGES_AND_DOCKERFILES"

  echo "All images pushed successfully."
}

main "$@"
