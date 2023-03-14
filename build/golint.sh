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

SKIP_DIR_REGEX="pkg/client"
TIMEOUT="10m"

echo "Running golangci-lint..."

golangci-lint run --timeout ${TIMEOUT} --skip-dirs ${SKIP_DIR_REGEX} -E govet,whitespace,gocognit,unparam -e '`ctx` is unused'

# gofmt should run everywhere, including
#   1. Skipped directories in previous step
#   2. Build exempted files using build tags
#      Note: Future build tags should be included.
echo "Running gofmt..."
golangci-lint run --timeout ${TIMEOUT} --disable-all --enable gofmt --build-tags integration

echo "PASS"
echo

echo "PASS"
echo
