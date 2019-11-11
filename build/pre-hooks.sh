#!/bin/bash

set -o errexit
set -o nounset
set -o xtrace

# Run the packr2 command.
# It will look for all the boxes in your code and then generate .go files that
# pack the static files into bytes that can be bundled into the Go binary.
# Remove auto generated files by packr2
pushd cmd/controller
packr2
popd
