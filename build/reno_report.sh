#!/bin/bash

version=${1:-"''"}

## Generate changelog files using reno

## Used tools:
## - reno for changelog generation
## - rst2md for report conversion from rst to markdown

## This script generates changelog files in markdown format.
## Generated files are:
## - CHANGELOG.md contains all versions history (for versions which have reno notes) 
## - CHANGELOG_CURRENT.md contains only current version changes

## Arguments:
## version ($1) - string to use for "current" version number  

## Latest notes should go to current version
## We generate report before tagging the repo, so we need to set this version here
# echo "unreleased_version_title: ${version}" > reno.yaml
sed "s/unreleased_version_title: ''/unreleased_version_title: ${version}/g" reno.yaml > releasenotes/config.yaml

# Update changelog for all versions:

## Generate rst report
echo reno report --output ./CHANGELOG.rst
reno report --output ./CHANGELOG.rst

## Convert rst to markdown
rst2md ./CHANGELOG.rst --output ./CHANGELOG.md

# Generate changelog for current version only:

## Reno `--version` flag does not support "unreleased" setting and requires specific version, even if it's dynamic
## To generate dynamic version, use `reno list`.
## It will be replaced by `unreleased_version_title` setting in the actual report file
UNRELEASED_VERSION=$(reno list 2>/dev/null | grep -E "^[0-9]+\.[0-9]+\.[0-9]+\-[0-9]+")

if [ -n "${UNRELEASED_VERSION}" ]
then
	## Generate rst report
	echo reno report --version=${UNRELEASED_VERSION} --output ./CHANGELOG_CURRENT.rst
	reno report --version=${UNRELEASED_VERSION} --output ./CHANGELOG_CURRENT.rst
	## Convert rst to markdown
	rst2md ./CHANGELOG_CURRENT.rst --output ./CHANGELOG_CURRENT.md
fi
