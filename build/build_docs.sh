#!/usr/bin/env bash

set -o errexit
set -o xtrace

cd docs

if [[ -z ${VERSION} ]]
then
echo "{\"version\":\"${VERSION}\"}" > ./.vitepress/version.json
fi

npm install -g pnpm@9

pnpm cache clean
pnpm install --force
pnpm docs:build
