#!/bin/bash

# Copyright 2021 The Kanister Authors.
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

IMAGE_REGISTRY="ghcr.io/kanisterio"

PUBLISHED_IMAGES_NAME_PATH="build/published_images.json"

TAG=${1:-"v9.99.9-dev"}

COMMIT_SHA_TAG=commit-${COMMIT_SHA:?"COMMIT_SHA is required"}
SHORT_COMMIT_SHA_TAG=short-commit-${COMMIT_SHA::12}

push_images() {
   images_file_path=$1

   images=$(jq -r .images[] "${images_file_path}")

   for i in ${images[@]}; do
      docker tag $IMAGE_REGISTRY/$i:$TAG $IMAGE_REGISTRY/$i:$COMMIT_SHA_TAG
      docker tag $IMAGE_REGISTRY/$i:$TAG $IMAGE_REGISTRY/$i:$SHORT_COMMIT_SHA_TAG
      docker push $IMAGE_REGISTRY/$i:$TAG
      docker push $IMAGE_REGISTRY/$i:$COMMIT_SHA_TAG
      docker push $IMAGE_REGISTRY/$i:$SHORT_COMMIT_SHA_TAG
   done
}

push_images $PUBLISHED_IMAGES_NAME_PATH
