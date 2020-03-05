#!/bin/bash

set -o errexit
set -o nounset
set -o xtrace

GOLANG_VERSIONS="1.14"
GOLANG_TAR="go${GOLANG_VERSIONS}.linux-amd64.tar.gz"
GOLANG_SHA="08df79b46b0adf498ea9f320a0f23d6ec59e9003660b4c9c1ce8e5e2c6f823ca"

GO_SWAGGER_VERSION="v0.22.0"
GO_SWAGGER_BIN="swagger_linux_amd64"
GO_SWAGGER_SHA="86427338df4d71062a89684acda338ce2064c451854fd4487eb354fec280aa57"

GOMETALINTER_VERSION="2.0.5"
GOMETALINTER_TGZ="gometalinter-${GOMETALINTER_VERSION}-linux-amd64.tar.gz"
GOMETALINTER_SHA="83ff1a03626130d249b96b7e321d9c7a03e5f943c042a0e07011779be1adf8e8"

GOLANGCI_LINT_VERSION="1.22.2"
GOLANGCI_LINT_TGZ="golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64.tar.gz"
GOLANGCI_LINT_SHA="109d38cdc89f271392f5a138d6782657157f9f496fd4801956efa2d0428e0cbe"

fetch_check() {
    local -r url="${1}"
    local -r file="${2}"
    local -r sha="${3}"
    echo "Feching ${file} ..." >&2
    wget --progress=dot:mega "${url}/${file}"
    echo "${sha}  ${file}" | sha256sum -c
}

echo "================= Installing Go tools ==================="

pushd /tmp

fetch_check https://storage.googleapis.com/golang "${GOLANG_TAR}" "${GOLANG_SHA}"
tar -C /usr/local -xvzf ${GOLANG_TAR}

fetch_check "https://github.com/go-swagger/go-swagger/releases/download/${GO_SWAGGER_VERSION}" \
    "${GO_SWAGGER_BIN}" "${GO_SWAGGER_SHA}"
chmod +x ${GO_SWAGGER_BIN}
mv ${GO_SWAGGER_BIN} /usr/local/bin/swagger

fetch_check "https://github.com/alecthomas/gometalinter/releases/download/v${GOMETALINTER_VERSION}" \
    "${GOMETALINTER_TGZ}" "${GOMETALINTER_SHA}"
tar -C /usr/local/bin --strip-components 1 -xvzf "${GOMETALINTER_TGZ}"

fetch_check "https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}" \
    "${GOLANGCI_LINT_TGZ}" "${GOLANGCI_LINT_SHA}"
tar -C /usr/local/bin --strip-components 1 -xvzf "${GOLANGCI_LINT_TGZ}"

popd

export GOROOT=/usr/local/go
export GOPATH=/go
export GOBIN=/usr/local/bin

${GOROOT}/bin/go get -u golang.org/x/lint/golint
${GOROOT}/bin/go get -u github.com/kisielk/errcheck
${GOROOT}/bin/go get -u gopkg.in/tebeka/go2xunit.v1

rm -rf ${GOPATH}

rm -rf /tmp && mkdir /tmp && chmod 1777 /tmp
