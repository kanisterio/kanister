# Copyright 2019 The Kanister Authors.
#
# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# include repository server's makefile
include Makefile.kubebuilder

# The binary to build (just the basename).
BIN := controller

# This repo's root import path (under GOPATH).
PKG := github.com/kanisterio/kanister

# Where to push the docker image.
REGISTRY ?= kanisterio

# Which architecture to build - see $(ALL_ARCH) for options.
ARCH ?= amd64

# This version-strategy uses git tags to set the version string
VERSION ?= $(shell git describe --tags --always --dirty)
#
# This version-strategy uses a manual value to set the version string
#VERSION := 1.2.3

PWD := $$(pwd)

# Whether to build inside a containerized build environment
DOCKER_BUILD ?= "true"

# Mention the vm-driver that should be used to install OpenShift
vm-driver ?= "kvm"

# Default OCP version in which the OpenShift apps are going to run
ocp_version ?= "4.13"
###
### These variables should not need tweaking.
###

SRC_DIRS := cmd pkg # directories which hold app source (not vendored)

ALL_ARCH := amd64 arm arm64 ppc64le

# Set default base image dynamically for each arch

IMAGE_NAME := $(BIN)

IMAGE := $(REGISTRY)/$(IMAGE_NAME)

DOCS_RELEASE_BUCKET ?= s3://docs.kanister.io

GITHUB_TOKEN ?= ""

GOEXPERIMENT ?= ""

## Tool Versions

CONTROLLER_TOOLS_VERSION ?= "v0.12.0"

## Changelog file for goreleaser

CHANGELOG_FILE ?= ./CHANGELOG_CURRENT.md

# If you want to build all binaries, see the 'all-build' rule.
# If you want to build all containers, see the 'all-container' rule.
# If you want to build AND push all containers, see the 'all-push' rule.
all: build

build-%:
	@$(MAKE) --no-print-directory ARCH=$* build

container-%:
	@$(MAKE) --no-print-directory ARCH=$* container

push-%:
	@$(MAKE) --no-print-directory ARCH=$* push

all-build: $(addprefix build-, $(ALL_ARCH))

all-container: $(addprefix container-, $(ALL_ARCH))

all-push: $(addprefix push-, $(ALL_ARCH))

build: bin/$(ARCH)/$(BIN)

build-controller:
	@$(MAKE) run CMD=" \
	GOOS=linux GOARCH=$(ARCH) GOEXPERIMENT=$(GOEXPERIMENT) goreleaser build --id $(BIN) --rm-dist --debug --snapshot --single-target \
	&& cp dist/$(BIN)_linux_$(ARCH)*/$(BIN) bin/$(ARCH)/$(BIN) \
	"

bin/$(ARCH)/$(BIN):
	@echo "building: $@"
	@$(MAKE) run CMD=" \
		GOARCH=$(ARCH)       \
		VERSION=$(VERSION) \
		PKG=$(PKG)         \
		BIN=$(BIN) \
		./build/build.sh   \
	"
# Example: make shell CMD="-c 'date > datefile'"
shell: build-dirs
	@echo "launching a shell in the containerized build environment"
	@PWD=$(PWD) ARCH=$(ARCH) PKG=$(PKG) GITHUB_TOKEN=$(GITHUB_TOKEN) CMD="/bin/bash $(CMD)" /bin/bash ./build/run_container.sh shell

DOTFILE_IMAGE = $(subst :,_,$(subst /,_,$(IMAGE))-$(VERSION))

container: .container-$(DOTFILE_IMAGE) container-name
.container-$(DOTFILE_IMAGE): bin/$(ARCH)/$(BIN) Dockerfile.in
	@/bin/bash -c "                   \
		BIN=$(BIN)                    \
		ARCH=$(ARCH)                  \
		IMAGE=$(IMAGE)                \
		VERSION=$(VERSION)            \
		SOURCE_BIN=bin/$(ARCH)/$(BIN) \
		./build/package.sh            \
	"

container-name:
	@echo "container: $(IMAGE):$(VERSION)"

release-controller: .push-$(DOTFILE_IMAGE) push-name
.push-$(DOTFILE_IMAGE): .container-$(DOTFILE_IMAGE)
ifeq ($(findstring gcr.io,$(REGISTRY)),gcr.io)
	@gcloud docker -- push $(IMAGE):$(VERSION)
else
	@docker push $(IMAGE):$(VERSION)
endif
	@docker images -q $(IMAGE):$(VERSION) > $@

push-name:
	@echo "pushed: $(IMAGE):$(VERSION)"

version:
	@echo $(VERSION)

.PHONY: deploy format-vet go-test test codegen build-dirs run clean container-clean bin-clean docs start-kind tiller stop-kind release-snapshot go-mod-download

deploy: release-controller .deploy-$(DOTFILE_IMAGE)
.deploy-$(DOTFILE_IMAGE):
	@sed                        \
		-e 's|IMAGE|$(IMAGE)|g' \
		-e 's|TAG|$(VERSION)|g' \
		bundle.yaml.in > .deploy-$(DOTFILE_IMAGE)
	@kubectl apply -f .deploy-$(DOTFILE_IMAGE)

format-vet: build-dirs
	@$(MAKE) run CMD="./build/format-vet.sh $(SRC_DIRS)"

go-test: build-dirs
	@$(MAKE) run CMD="TEST_FILTER=$(TEST_FILTER) ./build/test.sh $(SRC_DIRS)"

test: format-vet go-test

helm-test: build-dirs
	@$(MAKE) run CMD="./build/helm-test.sh $(SRC_DIRS)"

integration-test: build-dirs
	@$(MAKE) run CMD="./build/integration-test.sh short"

openshift-test:
	@/bin/bash ./build/integration-test.sh openshift $(ocp_version)

golint:
	@$(MAKE) run CMD="./build/golint.sh"

codegen:
	@$(MAKE) run CMD="./build/codegen.sh"

docs:
	@$(MAKE) run BUILD_IMAGE="node:20-slim" CMD="VERSION=$(VERSION) ./build/build_docs.sh"

API_DOCS_CMD = "gen-crd-api-reference-docs 			\
		-config docs/api_docs/config.json 	\
		-api-dir ./pkg/apis/cr/v1alpha1 	\
		-template-dir docs/api_docs/template 		\
		-out-file API.md"

crd_docs:
ifeq ($(DOCKER_BUILD),"true")
	@echo "running API_DOCS_CMD in the containerized build environment"
	@PWD=$(PWD) ARCH=$(ARCH) CMD=$(API_DOCS_CMD) /bin/bash ./build/run_container.sh crd_docs
else
	@/bin/bash -c $(API_DOCS_CMD)
endif

build-dirs:
	@mkdir -p bin/$(ARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(ARCH)

run: build-dirs
ifeq ($(DOCKER_BUILD),"true")
	@echo "running CMD in the containerized build environment"
	@PWD=$(PWD) ARCH=$(ARCH) PKG=$(PKG) GITHUB_TOKEN=$(GITHUB_TOKEN) CMD="$(CMD)" /bin/bash ./build/run_container.sh run
else
	@/bin/bash -c "$(CMD)"
endif

clean: dotfile-clean bin-clean

dotfile-clean:
	rm -rf .container-* .dockerfile-* .push-* .deploy-*

bin-clean:
	rm -rf .go bin

package-helm:
	@$(MAKE) run CMD="PACKAGE_FOLDER=${PACKAGE_FOLDER} HELM_RELEASE_REPO_URL=${HELM_RELEASE_REPO_URL} HELM_RELEASE_REPO_INDEX=${HELM_RELEASE_REPO_INDEX} ./build/package_helm.sh $(VERSION)"

gorelease:
	@$(MAKE) run CMD="CHANGELOG_FILE=$(CHANGELOG_FILE) GORELEASE_PARAMS=${GORELEASE_PARAMS}  GHCR_LOGIN_TOKEN=${GHCR_LOGIN_TOKEN} GHCR_LOGIN_USER=${GHCR_LOGIN_USER} ./build/gorelease.sh"

release-snapshot:
	@$(MAKE) run CMD="GOOS=linux GOARCH=$(ARCH) GORELEASER_CURRENT_TAG=v9.99.9-dev GOEXPERIMENT=$(GOEXPERIMENT) goreleaser --debug release --rm-dist --snapshot --timeout=60m0s"

go-mod-download:
	@$(MAKE) run CMD="go mod download"

start-kind:
	@$(MAKE) run CMD="./build/local_kubernetes.sh start_localkube"

tiller:
	@/bin/bash ./build/init_tiller.sh

install-minio:
	@$(MAKE) run CMD="./build/minio.sh install_minio"

install-csi-hostpath-driver:
	@$(MAKE) run CMD="./build/local_kubernetes.sh install_csi_hostpath_driver"

uninstall-minio:
	@$(MAKE) run CMD="./build/minio.sh uninstall_minio"

start-minishift:
	@/bin/bash ./build/minishift.sh start_minishift $(vm-driver)

stop-minishift:
	@/bin/bash ./build/minishift.sh stop_minishift

stop-kind:
	@$(MAKE) run CMD="./build/local_kubernetes.sh stop_localkube"

check:
	@./build/check.sh

go-mod-tidy:
	@$(MAKE) run CMD="./build/gomodtidy.sh"


install-crds: ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	@$(MAKE) run CMD="kubectl apply -f pkg/customresource/"

uninstall-crds: ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	@$(MAKE) run CMD="kubectl delete -f pkg/customresource/"

manifests: ## Generates CustomResourceDefinition objects.
	@$(MAKE) run CMD="./build/generate_crds.sh ${CONTROLLER_TOOLS_VERSION}"

reno-new:
	@PWD=$(PWD) ARCH=$(ARCH) PKG=$(PKG) GITHUB_TOKEN=$(GITHUB_TOKEN) CMD="EDITOR=vim reno new $(note) --edit" /bin/bash ./build/run_container.sh shell

reno-report:
	@$(MAKE) run CMD="./build/reno_report.sh $(VERSION)"

reno-lint:
	@$(MAKE) run CMD="reno lint"
