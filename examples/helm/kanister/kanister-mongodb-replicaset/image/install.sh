#!/bin/bash

set -o errexit
set -o nounset
set -o xtrace

mongo_deb="mongodb-consistent-backup_1.1.0-2_amd64.deb"

cd /kanister

echo "============================ Create a log file ================================="
touch /var/log/kanister.log

echo "================= Adding some global settings ==================="
mv gbl_env.sh /etc/profile.d/
mkdir -p ${HOME}/.ssh/
mv config ${HOME}/.ssh/
mv 90forceyes /etc/apt/apt.conf.d/

echo "================= Installing basic packages ==================="
apt-get update && \
apt-get install wget musl-dev python -y

echo "================= Installing Python packages ==================="
wget --progress=dot:mega https://bootstrap.pypa.io/get-pip.py
python get-pip.py

echo "================= Adding awscli ============"
pip install awscli

echo "================= Install Mongo Tools ==================="
wget --progress=dot:mega https://github.com/Percona-Lab/mongodb_consistent_backup/releases/download/1.1.0/${mongo_deb}
dpkg -i ./${mongo_deb}

echo "================= Cleaning package lists ==================="
apt-get clean
apt-get autoclean
apt-get autoremove
rm -f ./${mongo_deb}
rm -f get-pip.py
rm -rf /var/lib/apt/lists/*
