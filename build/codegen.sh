#!/bin/bash

# Copyright 2016 The Rook Authors. All rights reserved.
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

# This is a workaround for the fact that codegen requires this boilerplate file
# to exist in GOPATH. If this file isn't present, we'll use an empty one.
boilerplate="${GOPATH}/src/k8s.io/kubernetes/hack/boilerplate/boilerplate.go.txt"
mkdir -p $(dirname "${boilerplate}")
touch "${boilerplate}"

pushd vendor/k8s.io/code-generator
./generate-groups.sh                        \
  all                                       \
  github.com/kanisterio/kanister/pkg/client \
  github.com/kanisterio/kanister/pkg/apis   \
  "cr:v1alpha1"                             \
