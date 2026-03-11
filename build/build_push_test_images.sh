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
REPOSITORY=${LOCAL_IMAGE_REPOSITORY:-"test-images"}
KIND_CLUSTER=${KIND_CLUSTER_NAME:-"integration-testing-cluster"}

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
# Helpers
########################################

# Image naming: {ORG}/{REPOSITORY}/{app_name}:latest
# e.g. kanisterio/test-images/mongodb:latest
image_ref() {
  echo "${ORG}/${REPOSITORY}/${1}:latest"
}

# Build an image into the local Docker daemon, then immediately load it into
# Kind and remove it from the daemon to reclaim disk space.
build_load_clean() {
  local app_name="$1"
  local dockerfile="$2"
  local ref
  ref=$(image_ref "${app_name}")

  echo "→ Building ${ref} from ${dockerfile}"
  docker buildx build \
    --platform=linux/amd64 \
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
  echo "Image org:        ${ORG}"
  echo "Image repository: ${REPOSITORY}"
  echo "Kind cluster:     ${KIND_CLUSTER}"

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
    --platform=linux/amd64 \
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
    build_load_clean "${app_name}" "${dockerfile}"
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
