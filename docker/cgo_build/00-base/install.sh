#!/bin/bash

set -o errexit
set -o nounset
set -o xtrace

locale-gen en_US en_US.UTF-8 && dpkg-reconfigure locales

cd "$(dirname ${0})"

echo "================= Adding some global settings ==================="
mkdir -p ${HOME}/.ssh/
mv config ${HOME}/.ssh/
mv 90forceyes /etc/apt/apt.conf.d/
mv go_env.sh /etc/profile.d/

echo "================= Updating package lists ==================="
apt-get update

echo "================= Installing basic packages ==================="
apt-get install \
  sudo \
  vim \
  curl \
  wget \
  git \
  jq \
  unzip \
  apt-transport-https \
  rsync \
  libltdl-dev

echo "================= Installing Python packages ==================="
apt-get install \
  python-pip \
  software-properties-common \
  enchant

echo "================= Installing certificate authorities ==================="
apt-get install \
  ca-certificates
update-ca-certificates

echo "================= Installing shellcheck ============="
apt-get install shellcheck

echo "================= Cleaning package lists ==================="
apt-get clean
apt-get autoclean
apt-get autoremove

# Delete all the apt list files since they're big and get stale quickly.
rm -rf /var/lib/apt/lists/*
rm -rf /tmp && mkdir /tmp && chmod 1777 /tmp
