#!/bin/bash

set -o errexit
set -o nounset
set -o xtrace

cd /install

echo "================= Adding some global settings ==================="
mv gbl_env.sh /etc/profile.d/
mkdir -p ${HOME}/.ssh/
mv config ${HOME}/.ssh/

echo "================= Installing basic packages ==================="
apk add --update --no-cache \
        ca-certificates \
        bash curl groff less python py-pip \
 && update-ca-certificates \
 && rm -rf /var/cache/apk/* \
           /tmp/*

echo "================= Installing Python packages ==================="
pip install virtualenv
pip install --upgrade pip

echo "================= Adding awscli ============"
pip install awscli
