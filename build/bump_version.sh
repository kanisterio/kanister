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


usage() {
    echo ./build/bump_version.sh previous_versions release_version
    exit 1
}

main() {
    local prev=${1:?"$(usage)"}; shift
    local next=${1:?"$(usage)"}; shift
    echo $prev
    echo $next
    if [ "$#" -eq 0 ]; then
            pkgs=( docker/ scripts/ examples/ pkg/ helm/ )
        else
            pkgs=( "$@" )
    fi
    
    for ((i=${#prev[@]}-1; i>=0; i--)); do
        # -F matches for exact words, not regular expression (-E), that is what required here
        grep -F "${prev[$i]}" -Ir  "${pkgs[@]}" --exclude-dir={docs,mod,bin,html,frontend} --exclude=\*.sum --exclude=\*.mod | cut -d ':' -f 1 | uniq | xargs sed -ri "s/${prev[$i]}/${next//./\\.}/g"
    done

    # Modify the first instabnce of kanister_tools_version in docs/constants.ts
    array_size=${#pkgs[@]}
    if [ $array_size -eq 1 ]; then
      file_count=$(find "${pkgs[@]}/docs" -maxdepth 1 -name "constants.ts" | wc -l)
      if [ $file_count -eq 1 ]; then
          sed -i "0,/'kanister_tools_version': *,/s/'kanister_tools_version': '${next//./\\.}',/" "${pkgs[@]}/docs/constants.ts"
      fi
    fi
}

main $@
