#!/bin/sh

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

if [ -z "${BIN:-""}" ]; then
    echo "BIN must be set"
    exit 1
fi
if [ -z "${ARCH:-""}" ]; then
    echo "ARCH must be set"
    exit 1
fi
if [ -z "${BASEIMAGE:-""}" ]; then
    echo "BASEIMAGE must be set"
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

sed                                \
    -e "s|ARG_BIN|${BIN}|g"        \
    -e "s|ARG_ARCH|${ARCH}|g"      \
    -e "s|ARG_FROM|${BASEIMAGE}|g" \
    -e "s|ARG_SOURCE_BIN|${SOURCE_BIN}|g" \
    Dockerfile.in > .dockerfile-${ARCH}
docker build -t ${IMAGE}:${VERSION} -f .dockerfile-${ARCH} .
docker images -q ${IMAGE}:${VERSION}
