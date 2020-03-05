#!/bin/bash

set -o errexit
set -o nounset
set -o xtrace

KUBECTL_VERSION="v1.15.0"
KUBECTL_BIN="kubectl"
KUBECTL_SHA="ecec7fe4ffa03018ff00f14e228442af5c2284e57771e4916b977c20ba4e5b39"

HELM_3_VERSION="v3.1.0"

fetch_check() {
    local -r url="${1}"
    local -r file="${2}"
    local -r sha="${3}"
    echo "Feching ${file} ..." >&2
    wget --progress=dot:mega "${url}/${file}"
    echo "${sha}  ${file}" | sha256sum -c
}

fetch_check \
    "https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64" \
    "${KUBECTL_BIN}" "${KUBECTL_SHA}"
chmod +x "${KUBECTL_BIN}"
mv "${KUBECTL_BIN}" "/usr/local/bin/${KUBECTL_BIN}"

echo "================= Installing kubectl-aliases ==================="
pushd $HOME && curl -O https://raw.githubusercontent.com/ahmetb/kubectl-alias/master/.kubectl_aliases && popd
echo "source ~/.kubectl_aliases" >> $HOME/.bashrc

echo "================= Installing Helm 3 ==================="
wget --progress=dot:mega https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 -O get_helm.sh
bash get_helm.sh -v ${HELM_3_VERSION}
rm -f get_helm.sh
# mv /usr/local/bin/helm /usr/local/bin/helm3
helm plugin install https://github.com/helm/helm-2to3

echo "================= Installing kind ==================="
wget -O /usr/local/bin/kind https://github.com/kubernetes-sigs/kind/releases/download/v0.5.1/kind-linux-amd64
chmod +x /usr/local/bin/kind

rm -rf /tmp && mkdir /tmp && chmod 1777 /tmp
