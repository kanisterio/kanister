#!/usr/bin/env bash
# Copyright 2019 The Kanister Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

usage () {
    echo ./build/managetags.sh remote_name tag_name
    exit 1
}

# deletes a git tag locally as well as from remote if it is already persent
main() {
    local remote=${1:?"$(usage)"}
    local tag=${2:?"$(usage)"}

    if [ $(git tag -l ${tag}) ] ; then
        git tag -d ${tag}
        git push --delete ${remote} ${tag}
    fi
}

main $@
