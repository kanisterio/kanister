#!/usr/bin/env bash

# Copyright 2019 The Kanister Authors.
# This script is based off of the install script from helm, licensed under the
# Apache License, Version 2.0. The script was found here
# https://github.com/kubernetes/helm/blob/master/scripts/get

set -o errexit
set -o nounset
set -o xtrace
set -o pipefail

DIST_NAME="kanister"
BIN_NAMES=("kanctl")
RELEASES_URL="https://github.com/kanisterio/kanister/releases"

: ${KANISTER_INSTALL_DIR:="/usr/local/bin"}

# initArch discovers the architecture for this system.
initArch() {
    ARCH=$(uname -m)
    case $ARCH in
        armv5*) ARCH="armv5";;
        armv6*) ARCH="armv6";;
        armv7*) ARCH="armv7";;
        aarch64) ARCH="arm64";;
        x86) ARCH="386";;
        x86_64) ARCH="amd64";;
        i686) ARCH="386";;
        i386) ARCH="386";;
    esac
}

# initOS discovers the operating system for this system.
initOS() {
    OS=$(uname | tr '[:upper:]' '[:lower:]')
    case "$OS" in
        # On linux we also support kando
        linux) BIN_NAMES=("kanctl" "kando");;
        # Minimalist GNU for Windows
        mingw*) OS='windows';;
    esac
}

# runs the given command as root (detects if we are root already)
runAsRoot() {
    local cmd="$*"
    if [ $EUID -ne 0 ]; then
        cmd="sudo ${cmd}"
    fi
    ${cmd}
}

# verifySupported checks that the os/arch combination is supported for
# binary builds.
verifySupported() {
    local supported="\ndarwin-amd64\nlinux-amd64\nwindows-amd64"
    if ! echo "${supported}" | grep -q "${OS}-${ARCH}"; then
        echo "No prebuilt binary for ${OS}-${ARCH}."
        echo "To build from source, go to https://github.com/kanisterio/kanister"
        exit 1
    fi

    local required_tools=("curl" "shasum")
    for tool in "${required_tools[@]}"; do
        if ! type "${tool}" > /dev/null; then
            echo "${tool} is required"
            exit 1
        fi
    done
}

# checkDesiredVersion checks if the desired version is available.
checkDesiredVersion() {
    local version="${1}"
    # Use the GitHub releases webpage for the project to find the desired version for this project.
    local release_url="${RELEASES_URL}/tag/${version}"
    local tag=$(curl -SsL ${release_url} | awk '/\/tag\//' | grep -v no-underline | cut -d '"' -f 2 | awk '{n=split($NF,a,"/");print a[n]}' | awk 'a !~ $0{print}; {a=$0}')
    if [ "x${tag}" == "x" ]; then
        echo "Version tag ${version} not found."
        exit 1
    fi
}

# downloadFile downloads the binary and verifies the checksum.
downloadFile() {
    local version="${1}"

    local release_url="${RELEASES_URL}/download/${version}"
    local kanister_dist="${DIST_NAME}_${version}_${OS}_${ARCH}.tar.gz"
    local kanister_checksum="checksums.txt"

    local download_url="${release_url}/${kanister_dist}"
    local checksum_url="${release_url}/${kanister_checksum}"

    KANISTER_TMP_ROOT="$(mktemp -dt kanister-installer-XXXXXX)"
    KANISTER_TMP_FILE="${KANISTER_TMP_ROOT}/${kanister_dist}"
    kanister_sum_file="${KANISTER_TMP_ROOT}/${kanister_checksum}"

    echo "Downloading $download_url"
    curl -SsL "${checksum_url}" -o "${kanister_sum_file}"
    curl -SsL "${download_url}" -o "$KANISTER_TMP_FILE"

    echo "Checking hash of ${kanister_dist}"
    pushd "${KANISTER_TMP_ROOT}"
    local filtered_checksum="./${kanister_dist}.sha256"
    grep "${kanister_dist}" < "${kanister_checksum}" > "${filtered_checksum}"
    shasum -a 256 -c "${filtered_checksum}"
    popd
}

# installFile verifies the SHA256 for the file, then unpacks and
# installs it.
installFile() {
    pushd "${KANISTER_TMP_ROOT}"
    tar xvf "${KANISTER_TMP_FILE}"
    rm "${KANISTER_TMP_FILE}"
    echo "Preparing to install into ${KANISTER_INSTALL_DIR}"
    for bin_name in "${BIN_NAMES[@]}"; do
        runAsRoot cp "./${bin_name}" "${KANISTER_INSTALL_DIR}"
    done
    popd
}

# testBinaries tests the installed binaries make sure they're working.
testBinaries() {
    for bin_name in "${BIN_NAMES[@]}"; do
        echo "${bin_name} installed into ${KANISTER_INSTALL_DIR}/${bin_name}"
        if ! type "${bin_name}" > /dev/null; then
            echo "${bin_name} not found. Is ${KANISTER_INSTALL_DIR} on your PATH?"
            exit 1
        fi
    done
}

cleanup() {
  if [[ -d "${KANISTER_TMP_ROOT:-}" ]]; then
    rm -rf "$KANISTER_TMP_ROOT"
  fi
}

main() {
    version="${1:-"0.110.0"}"
    initArch
    initOS
    verifySupported
    checkDesiredVersion "${version}"
    downloadFile "${version}"
    installFile
    testBinaries
    cleanup
}

main $@

