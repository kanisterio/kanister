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

export PATH=$GOPATH/bin:$GOROOT/bin:$PATH

export CGO_ENABLED=0
export GO111MODULE=on

TARGETS=$(for d in "$@"; do echo ./$d/...; done)
TAGS=""

if [[ -n "${TEST_INTEGRATION+x}" ]]; then
    TAGS="-tags=integration -timeout 30m"
fi

echo "Running tests:"
go test -v ${TAGS} -installsuffix "static" -i ${TARGETS}
go test -v ${TAGS} ${TARGETS} -list .
go test -v ${TAGS} -installsuffix "static" ${TARGETS} -check.v
echo

echo -n "Checking gofmt: "
ERRS=$(find "$@" -type f -name \*.go | xargs gofmt -l 2>&1 || true)
if [ -n "${ERRS}" ]; then
    echo "FAIL - the following files need to be gofmt'ed:"
    for e in ${ERRS}; do
        echo "    $e"
    done
    echo
    exit 1
fi
echo "PASS"
echo

echo -n "Checking go vet: "
ERRS=$(go vet ${TARGETS} 2>&1 || true)
if [ -n "${ERRS}" ]; then
    echo "FAIL"
    echo "${ERRS}"
    echo
    # TODO: Renable govet. Currently generated code fails to pass go vet. report,
    # but don't exit on failures.
    #exit 1
fi
echo "PASS"
echo
