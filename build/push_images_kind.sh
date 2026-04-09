#!/bin/bash

# Copyright 2026 The Kanister Authors.
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

# IMAGE_REPOSITORY may include a repository path component (e.g. "kanisterio/test-images")
# when images are kind-loaded without a registry prefix.
IMAGE_REPOSITORY=${IMAGE_REPOSITORY:-"kanisterio/test-images"}
IMAGE_REGISTRY=${IMAGE_REGISTRY:-""}
IMAGE_TAG=${IMAGE_TAG:-"v9.99.9-dev"}
KIND_CLUSTER=${KIND_CLUSTER_NAME:-"kanister"}
PLATFORM="linux/${ARCH:-amd64}"

########################################
# App image matrix (tools handled separately as a build dependency)
# Format: app_name|dockerfile
########################################

APP_IMAGES="
mongodb|docker/mongodb/Dockerfile
postgresql|docker/postgresql/Dockerfile
mssql-tools|docker/mssql-tools/Dockerfile
mysql-sidecar|docker/kanister-mysql/image/Dockerfile
postgres-kanister-tools|docker/postgres-kanister-tools/Dockerfile
"

########################################
# Pre-requisite checks
########################################

check_prerequisites() {
  if ! command -v kind &>/dev/null; then
    echo "ERROR: 'kind' is not installed or not in PATH" >&2
    exit 1
  fi

  if ! kind get clusters 2>/dev/null | grep -qx "${KIND_CLUSTER}"; then
    echo "ERROR: Kind cluster '${KIND_CLUSTER}' not found. Available clusters:" >&2
    kind get clusters >&2 || true
    exit 1
  fi
}

########################################
# Helpers
########################################

# Image naming: [{IMAGE_REGISTRY}/]{IMAGE_REPOSITORY}/{app_name}:{IMAGE_TAG}
# e.g. kanisterio/test-images/mongodb:v9.99.9-dev  (no registry, kind load)
#      localhost:5001/kanisterio/mongodb:v9.99.9-dev (local registry)
image_ref() {
  if [[ -n "${IMAGE_REGISTRY}" ]]; then
    echo "${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}/${1}:${IMAGE_TAG}"
  else
    echo "${IMAGE_REPOSITORY}/${1}:${IMAGE_TAG}"
  fi
}

# Build an image into the local Docker daemon, then immediately load it into
# Kind and remove it from the daemon to reclaim disk space.
build_and_load() {
  local app_name="$1"
  local dockerfile="$2"
  local ref
  ref=$(image_ref "${app_name}")

  echo "→ Building ${ref} from ${dockerfile}"
  docker buildx build \
    --platform="${PLATFORM}" \
    --load \
    --no-cache \
    -f "${dockerfile}" \
    --build-arg "TOOLS_IMAGE=${TOOLS_IMAGE:-}" \
    -t "${ref}" \
    . 2>&1 | sed "s|^|[${app_name}] |"

  echo "→ Loading ${ref} into Kind cluster '${KIND_CLUSTER}'"
  kind load docker-image "${ref}" --name "${KIND_CLUSTER}"

  echo "→ Removing ${ref} from Docker daemon (image is now in Kind)"
  docker rmi "${ref}"
}

########################################
# Main
########################################

main() {
  check_prerequisites

  echo "Image registry: ${IMAGE_REGISTRY:-"(none)"}"
  echo "Image repository: ${IMAGE_REPOSITORY}"
  echo "Image tag:      ${IMAGE_TAG}"
  echo "Kind cluster:   ${KIND_CLUSTER}"

  ########################################
  # 1. Build kanister-tools first.
  #
  # Several app Dockerfiles do:  FROM ${TOOLS_IMAGE} AS tools_image
  # so the tools image must stay in the local Docker daemon for the duration of
  # the app builds.  It is loaded into Kind and cleaned up last (step 3) because
  # blueprints also reference it as a runtime image.
  ########################################

  local tools_app="kanister-tools"
  TOOLS_IMAGE=$(image_ref "${tools_app}")

  echo "→ Building ${TOOLS_IMAGE}"
  docker buildx build \
    --platform="${PLATFORM}" \
    --load \
    --no-cache \
    -f "docker/tools/Dockerfile" \
    -t "${TOOLS_IMAGE}" \
    . 2>&1 | sed "s|^|[kanister-tools] |"

  ########################################
  # 2. Build, load into Kind, and clean each app image.
  #
  # Processing images one-at-a-time keeps peak Docker daemon storage minimal:
  # only the tools image + one app image exist in the daemon at any moment.
  ########################################

  while IFS='|' read -r app_name dockerfile; do
    [[ -n "${app_name}" ]] || continue
    build_and_load "${app_name}" "${dockerfile}"
  done <<< "${APP_IMAGES}"

  ########################################
  # 3. Load kanister-tools into Kind now that no further builds depend on it,
  #    then remove it from the Docker daemon.
  ########################################

  echo "→ Loading ${TOOLS_IMAGE} into Kind cluster '${KIND_CLUSTER}'"
  kind load docker-image "${TOOLS_IMAGE}" --name "${KIND_CLUSTER}"
  echo "→ Removing ${TOOLS_IMAGE} from Docker daemon (image is now in Kind)"
  docker rmi "${TOOLS_IMAGE}"

  echo "All images loaded into Kind successfully."
}

main "$@"
