#!/bin/sh

GO_BIN=go
DOCKER_BIN=docker
KUBECTL_BIN=kubectl

ok="✅"
warning="⚠️"
failed="❌"

pad=$(printf '%0.1s' "."{1..25})
pad_len=25

output() {
  local bin=$(echo "$1" | awk '{print toupper(substr( $0, 1, 1 )) substr($0, 2)}')
  local result=$2
  local msg=$(echo "$3" | awk '{print toupper(substr( $0, 1, 1 )) substr($0, 2)}')

  printf "* need %s" "${bin}"
  printf "%*.*s %s" 0 $((pad_len - ${#bin})) "${pad}" "${result}"
  if [ ! -z "${msg}" ]; then
    printf "\n → %s" "${msg}"
  fi
  printf "\n"
}

check_version_go() {
  local result=""

  if command -v ${GO_BIN} > /dev/null 2>&1 ; then
    result="${ok}"
  else
    result="${failed}"
    msg="${GO_BIN} not installed"
  fi

  output "${GO_BIN}" "${result}" "${msg}"
}


check_version_docker() {
  local result=""
  if command -v ${DOCKER_BIN} > /dev/null 2>&1 ; then
    result="${ok}"
  else
    result="${failed}"
    msg="${DOCKER_BIN} not installed"
  fi

  output "${DOCKER_BIN}" "${result}" "${msg}"
}

check_version_kubectl() {
  local result=""
  if command -v ${KUBECTL_BIN} > /dev/null 2>&1 ; then
    result="${ok}"
  else
    result="${failed}"
    msg="${KUBECTL_BIN} not installed"
  fi

  output "${KUBECTL_BIN}" "${result}" "${msg}"
}

check_version_go
check_version_docker
check_version_kubectl
