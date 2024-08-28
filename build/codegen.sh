#!/bin/bash

# Copyright 2019 The Kanister Authors.
#
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
set -o xtrace

export GO111MODULE=on
go mod download
execDir="/go/pkg/mod/k8s.io/code-generator@$(go list -f '{{.Version}}' -m k8s.io/code-generator)"
chmod +x "${execDir}"/generate-groups.sh
chmod +x "${execDir}"/generate-internal-groups.sh
"${execDir}"/generate-groups.sh                         \
  all                                        \
  github.com/kanisterio/kanister/pkg/client  \
  github.com/kanisterio/kanister/pkg/apis    \
  "cr:v1alpha1"                              \
  --go-header-file "$PWD"/build/boilerplate.go.txt \
  -o /go/src/
