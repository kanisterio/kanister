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

PWD="${PWD:-$(pwd)}"

# tag 0.1.0 is, 0.0.1 (latest) + gh + aws + helm binary
DOCS_BUILD_IMAGE="${DOCS_BUILD_IMAGE:-ghcr.io/kanisterio/docker-sphinx:0.2.0}"
BUILD_IMAGE="${BUILD_IMAGE:-ghcr.io/kanisterio/build:v0.0.24}"
PKG="${PKG:-github.com/kanisterio/kanister}"

ARCH="${ARCH:-amd64}"
PLATFORM="linux/${ARCH}"

check_param() {
  local arg_name=${1}
  local value_of_arg=""
  eval value_of_arg=\$$arg_name
  if [ -z "${value_of_arg}" ]; then
      echo "$arg_name must be set"
      exit 1
  fi
}


run_build_container() {
  local github_token="${GITHUB_TOKEN:-}"
  local extra_params="${EXTRA_PARAMS:-}"
  local cmd="${CMD:-command_is_not_set}"

  docker run                                                      \
      --platform ${PLATFORM}                                      \
      ${extra_params}                                             \
      --rm                                                        \
      --net host                                                  \
      -e GITHUB_TOKEN="${github_token}"                           \
      -v "${HOME}/.kube:/root/.kube"                              \
      -v "${PWD}/.go/pkg:/go/pkg"                                 \
      -v "${PWD}/.go/cache:/go/.cache"                            \
      -v "${PWD}:/go/src/${PKG}"                                  \
      -v "${PWD}/bin/${ARCH}:/go/bin"                             \
      -v "${PWD}/.go/std/${ARCH}:/usr/local/go/pkg/linux_${ARCH}" \
      -v "${HOME}/.docker:/root/.docker"                         \
      -v /var/run/docker.sock:/var/run/docker.sock                \
      -w /go/src/${PKG}                                           \
      ${BUILD_IMAGE}                                              \
      ${cmd}
}

run_docs_container() {
  check_param "IMAGE"
  check_param "CMD"

  docker run             \
    		--platform ${PLATFORM} \
    		--entrypoint ''     \
    		--rm                \
    		-v "${PWD}:/repo"   \
    		-w /repo            \
    		${IMAGE} \
    		${CMD}
}

build() {
  check_param "CMD"

  echo "Running build container..."
  run_build_container
}

shell() {
  echo "Running build container in interactive shell mode..."
  EXTRA_PARAMS="-ti" CMD="/bin/bash" run_build_container
}

docs() {
  check_param "CMD"

  echo "Running docs container..."
  IMAGE=${DOCS_BUILD_IMAGE} run_docs_container
}

crd_docs() {
  check_param "CMD"

  echo "Running crd docs container..."
  IMAGE=${BUILD_IMAGE} run_docs_container
}

usage() {
    cat <<EOM
Usage: ${0} <operation>
Where operation is one of the following:
  build: run some command within a build container
  crd_docs: build API docs within a container
  docs: build docs within a container
  shell: run build container in interactive shell mode
EOM
    exit 1
}

[ ${#@} -gt 0 ] || usage
case "${1}" in
        # Alphabetically sorted
        build)
            time -p build
            ;;
        crd_docs)
            time -p crd_docs
            ;;
        docs)
            time -p docs
            ;;
        shell)
            time -p shell
            ;;
        *)
            usage
            exit 1
esac

