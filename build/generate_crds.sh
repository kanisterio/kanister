#!/bin/bash

# Copyright 2023 The Kanister Authors.
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

## Location to install dependencies to
LOCALBIN=$(pwd)/bin
## Tool Binaries
CONTROLLER_GEN=${LOCALBIN}/controller-gen
## Tool Versions
CONTROLLER_TOOLS_VERSION=${1}

test -s ${CONTROLLER_GEN} || GOBIN=${LOCALBIN} go install sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_TOOLS_VERSION}
${CONTROLLER_GEN} crd webhook paths="github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1" output:crd:artifacts:config=pkg/customresource
