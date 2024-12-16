#!/usr/bin/env bash

set -o errexit
set -o xtrace

cd docs_new

if [[ -z ${VERSION} ]]
then
echo "{\"version\":\"${VERSION}\"}" > ./.vitepress/version.json
fi

npm install -g pnpm

pnpm cache clean
pnpm install --force
pnpm docs:build
