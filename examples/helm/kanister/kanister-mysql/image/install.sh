#!/bin/bash

set -o errexit
set -o nounset
set -o xtrace

cd /kanister

echo "============================ Create a log file ================================="
touch /var/log/kanister.log

echo "================= Adding some global settings ==================="
mv gbl_env.sh /etc/profile.d/
mkdir -p ${HOME}/.ssh/
mv config ${HOME}/.ssh/
mv 90forceyes /etc/apt/apt.conf.d/

echo "================= Installing basic packages ==================="
apt-get update
apt-get install curl sudo wget groff

echo "================= Install Mysql Tools ==================="
apt-get install mysql-client

echo "================= Installing Python packages ==================="
apt-get install \
  python-pip \
  python-software-properties \
  python-dev
pip install virtualenv

echo "================= Adding awscli ============"
pip install awscli

echo "================= Adding gcloud ============"
# Working around https://bugs.launchpad.net/ubuntu/+source/apt/+bug/1583102
chown root:root /tmp && chmod 1777 /tmp
# Copied from https://cloud.google.com/sdk/docs/quickstart-debian-ubuntu
export CLOUD_SDK_REPO="cloud-sdk-$(lsb_release -c -s)"
echo "deb http://packages.cloud.google.com/apt $CLOUD_SDK_REPO main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
apt-get update
apt-get install google-cloud-sdk

echo "================= Cleaning package lists ==================="
apt-get clean
apt-get autoclean
apt-get autoremove
