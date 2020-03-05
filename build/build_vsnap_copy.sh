#!/bin/sh -x

# Copyright 2020 The Kanister Authors.
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

if [ -z "${PKG}" ]; then
    echo "PKG must be set"
    exit 1
fi
if [ -z "${VERSION}" ]; then
    echo "VERSION must be set"
    exit 1
fi

# gcc may not be installed
which gcc >/dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "gcc not found"
    exit 1
fi

export GOPATH=/go
export GO111MODULE=on
SRC=/go/src

# set up an alternate go.mod file (needs go v1.14+)
ALT_MOD=./go_vsnap.mod
ALT_SUM=./go_vsnap.sum
REPLINE1='github.com/vmware-tanzu/astrolabe => /go/src/github.com/vmware-tanzu/astrolabe'
REPLINE2='github.com/vmware/gvddk => /go/src/github.com/vmware/gvddk'
REQLINE1='github.com/vmware-tanzu/astrolabe v0.0.0-00010101000000-000000000000'
REQLINE2='github.com/vmware/gvddk v0.0.0-00010101000000-000000000000'
sed \
    -e '/replace (/ a \\t'"${REPLINE1}"'\n\t'"${REPLINE2}" \
    -e '/require (/ a \\t'"${REQLINE1}"'\n\t'"${REQLINE2}" \
    go.mod >${ALT_MOD}
cp go.sum ${ALT_SUM}

# set up the local use of astrolabe (referenced from ALT_MOD)
mkdir -p $SRC/github.com/vmware-tanzu
cp -R /opt/vmware/astrolabe $SRC/github.com/vmware-tanzu
(cd $SRC/github.com/vmware-tanzu/astrolabe; go mod init)

# set up the local use of gvddk (referenced from ALT_MOD)
mkdir -p $SRC/github.com/vmware/gvddk
cp -R /opt/vmware/astrolabe/vendor/github.com/vmware/gvddk $SRC/github.com/vmware
(cd $SRC/github.com/vmware/gvddk; go mod init)

export CGO_ENABLED=1
export GO_EXTLINK_ENABLED=1
export CGO_LDFLAGS="-L/opt/vddk/lib64 -lvixDiskLib"
go install -v -modfile ${ALT_MOD} \
    -installsuffix "static" \
    -ldflags "-X ${PKG}/pkg/version.VERSION=${VERSION}" \
    ./cgo_cmd/vsnap_copy

# To run with cgo_shell:
#  LD_LIBRARY_PATH=/opt/vddk/lib64 bin/amd64/vsnap_copy
