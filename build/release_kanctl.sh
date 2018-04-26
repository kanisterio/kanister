#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o xtrace
set -o pipefail

if [ -z ${GITHUB_TOKEN} ]
then
	echo "Please set your GITHUB_TOKEN."
	echo "You can generate a token here: https://github.com/settings/tokens/new"
	exit 1
fi
goreleaser --rm-dist
