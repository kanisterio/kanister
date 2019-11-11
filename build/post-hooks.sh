#!/bin/bash

set -o errexit
set -o nounset
set -o xtrace

# Remove auto generated files by packr2
pushd cmd/controller
packr2 clean
popd
