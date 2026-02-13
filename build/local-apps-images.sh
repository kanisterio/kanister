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

IMAGE_TAG="${IMAGE_TAG:-dev}"

build_local_apps_images() {
  echo "Building local app images..."

  IMAGE_TAG="${IMAGE_TAG:-dev}"

  MAX_JOBS=3
  running_jobs=0
  failed=0

  TOOLS_IMAGE="kanisterio/kanister-tools:${IMAGE_TAG}"
  TOOLS_DOCKERFILE="docker/tools/Dockerfile"

  # ---- Step 1: Build tools image first ----
  echo "→ Building tools image ${TOOLS_IMAGE}"

  if [ ! -f "${TOOLS_DOCKERFILE}" ]; then
    echo "❌ Tools Dockerfile not found: ${TOOLS_DOCKERFILE}"
    exit 1
  fi

  if ! docker buildx build --platform=linux/amd64 --load --no-cache \
        -f "${TOOLS_DOCKERFILE}" \
        -t "${TOOLS_IMAGE}" \
        .; then
    echo "❌ Failed to build tools image"
    exit 1
  fi

  echo "✅ Tools image built"
  echo "Building remaining images (max ${MAX_JOBS} concurrent)..."

  # ---- Step 2: Parallel build remaining images ----
  while IFS='|' read -r image dockerfile; do
    [ -n "$image" ] || continue

    if [ ! -f "$dockerfile" ]; then
      echo "❌ Dockerfile not found: $dockerfile"
      exit 1
    fi

    echo "→ [START] ${image}:${IMAGE_TAG}"

    (
      docker buildx build --platform=linux/amd64 --load --no-cache \
        -f "$dockerfile" \
        --build-arg "TOOLS_IMAGE=${TOOLS_IMAGE}" \
        -t "${image}:${IMAGE_TAG}" \
        . 2>&1 | sed "s|^|[${image}] |"
    ) &


    running_jobs=$((running_jobs + 1))

    if [ "$running_jobs" -ge "$MAX_JOBS" ]; then
      if ! wait -n; then
        failed=1
      fi
      running_jobs=$((running_jobs - 1))
    fi

  done <<EOF
$IMAGES_AND_DOCKERFILES
EOF

  # ---- Step 3: Wait for remaining jobs ----
  while [ "$running_jobs" -gt 0 ]; do
    if ! wait -n; then
      failed=1
    fi
    running_jobs=$((running_jobs - 1))
  done

  if [ "$failed" -ne 0 ]; then
    echo "❌ One or more builds failed (see logs above)"
    exit 1
  fi

  echo "🎉 All images built successfully"
}

push_local_apps_images() {
  echo "Checking local registry availability at $REGISTRY_ADDR..."

  if ! curl -sf "http://$REGISTRY_ADDR/v2/" >/dev/null; then
    echo ""
    echo "❌ Local registry is not reachable at $REGISTRY_ADDR"
    echo ""
    echo "Install and expose the local registry using:"
    echo "  make install-registry"
    echo "  make expose-registry"
    echo ""
    exit 1
  fi

  echo "✅ Local registry is reachable"
  echo "Pushing images to local registry..."

  echo "$IMAGES_AND_DOCKERFILES" | while IFS='|' read -r image dockerfile; do
    [ -n "$image" ] || continue

    echo "→ Tagging and pushing ${image}:${IMAGE_TAG}"

    docker tag \
      "${image}:${IMAGE_TAG}" \
      "${REGISTRY_ADDR}/${image}:${IMAGE_TAG}"

    docker push "${REGISTRY_ADDR}/${image}:${IMAGE_TAG}"
  done

  echo "All images pushed successfully."
}


case "${1}" in
  build_local_apps_images)
    build_local_apps_images
    ;;
  push_local_apps_images)
    push_local_apps_images
    ;;
  *)
    echo "Usage: $0 {build_local_apps_images|push_local_apps_images}"
    exit 1
    ;;
esac

