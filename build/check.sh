#!/bin/bash

GO_BIN=go
DOCKER_BIN=docker
KUBECTL_BIN=kubectl

ok="✅"
warning="⚠️"
failed="❌"

pad=$(printf '%0.1s' "."{1..25})
pad_len=25

function output() {
  local bin=$1
  local result=$2
  local msg=$3

  printf "* need %s" "${bin^}"
  printf "%*.*s %s" 0 $((pad_len - ${#bin})) "${pad}" "${result}"
  if [ ! -z "${msg}" ]; then
    printf "\n → %s" "${msg}"
  fi
  printf "\n"
}

function check_version_go() {
  local expected=$(cat go.mod | grep "go [0-9]\.[0-9]*" | cut -d" " -f2)
  local result=""
  local msg=""

  if command -v ${GO_BIN} > /dev/null 2>&1 ; then
    local installed=$(go version | cut -d" " -f3)
    installed=${installed:2}

    if [ "${expected}" != "${installed}" ]; then
      result="${warning}"
      msg="version mismatched - got ${installed}, need ${expected}"
    else
      result="${ok}"
    fi
  else
    result="${failed}"
    msg="${GO_BIN^} not installed"
  fi

  output "${GO_BIN}" "${result}" "${msg}"
}

function check_version_docker() {
  local result=""
  if command -v ${DOCKER_BIN} > /dev/null 2>&1 ; then
    result="${ok}"
  else
    result="${failed}"
    msg="${DOCKER_BIN^} not installed"
  fi

  output "${DOCKER_BIN}" "${result}" "${msg}"
}

function check_version_kubectl() {
  local result=""
  if command -v ${KUBECTL_BIN} > /dev/null 2>&1 ; then
    result="${ok}"
  else
    result="${failed}"
    msg="${KUBECTL_BIN} not installed"
  fi

  output "${KUBECTL_BIN^}" "${result}" "${msg}"
}

check_version_go
check_version_docker
check_version_kubectl
