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

# The binary to build (just the basename).
BIN := controller

# This repo's root import path (under GOPATH).
PKG := github.com/kanisterio/kanister

# Where to push the docker image.
REGISTRY ?= kanisterio

# Which architecture to build - see $(ALL_ARCH) for options.
ARCH ?= amd64

# This version-strategy uses git tags to set the version string
VERSION := $(shell git describe --tags --always --dirty)
#
# This version-strategy uses a manual value to set the version string
#VERSION := 1.2.3

PWD := $$(pwd)

# Whether to build inside a containerized build environment
DOCKER_BUILD ?= "true"

DOCKER_CONFIG ?= "$(HOME)/.docker"

###
### These variables should not need tweaking.
###

SRC_DIRS := cmd pkg # directories which hold app source (not vendored)

INTEGRATION_TEST_DIR := pkg/testing # directory which hold workflow tests

ALL_ARCH := amd64 arm arm64 ppc64le

# Set default base image dynamically for each arch
ifeq ($(ARCH),amd64)
    BASEIMAGE?=alpine
endif
ifeq ($(ARCH),arm)
    BASEIMAGE?=armel/busybox
endif
ifeq ($(ARCH),arm64)
    BASEIMAGE?=aarch64/busybox
endif
ifeq ($(ARCH),ppc64le)
    BASEIMAGE?=ppc64le/busybox
endif

IMAGE_NAME := $(BIN)

IMAGE := $(REGISTRY)/$(IMAGE_NAME)

BUILD_IMAGE ?= kanisterio/build:v0.0.5
DOCS_BUILD_IMAGE ?= kanisterio/docker-sphinx

DOCS_RELEASE_BUCKET ?= s3://docs.kanister.io

GITHUB_TOKEN ?= ""

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

bin/$(ARCH)/$(BIN):
	@echo "building: $@"
	@$(MAKE) run CMD='-c " \
		GOARCH=$(ARCH)       \
		VERSION=$(VERSION) \
		PKG=$(PKG)         \
		./build/build.sh   \
	"'
# Example: make shell CMD="-c 'date > datefile'"
shell: build-dirs
	@echo "launching a shell in the containerized build environment"
	@docker run                                      \
		-ti                                          \
		--rm                                         \
		--privileged                                 \
		--net host                                   \
		-v "$(PWD)/.go/pkg:/go/pkg"                  \
		-v "$(PWD)/.go/cache:/go/.cache"             \
		-v "${HOME}/.kube:/root/.kube"               \
		-v "$(PWD):/go/src/$(PKG)"                   \
		-v "$(PWD)/bin/$(ARCH):/go/bin"              \
		-v "$(DOCKER_CONFIG):/root/.docker"          \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /go/src/$(PKG)                            \
		$(BUILD_IMAGE)                               \
		/bin/sh

DOTFILE_IMAGE = $(subst :,_,$(subst /,_,$(IMAGE))-$(VERSION))

container: .container-$(DOTFILE_IMAGE) container-name
.container-$(DOTFILE_IMAGE): bin/$(ARCH)/$(BIN) Dockerfile.in
	@/bin/bash -c "                   \
		BIN=$(BIN)                    \
		ARCH=$(ARCH)                  \
		BASEIMAGE=$(BASEIMAGE)        \
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

.PHONY: deploy test codegen build-dirs run clean container-clean bin-clean docs start-kind tiller stop-kind release-snapshot go-mod-download

deploy: release-controller .deploy-$(DOTFILE_IMAGE)
.deploy-$(DOTFILE_IMAGE):
	@sed                        \
		-e 's|IMAGE|$(IMAGE)|g' \
		-e 's|TAG|$(VERSION)|g' \
		bundle.yaml.in > .deploy-$(DOTFILE_IMAGE)
	@kubectl apply -f .deploy-$(DOTFILE_IMAGE)

test: build-dirs
	@$(MAKE) run CMD='-c "./build/test.sh $(SRC_DIRS)"'

integration-test: build-dirs
	@$(MAKE) run CMD='-c "TEST_INTEGRATION=true ./build/test.sh $(INTEGRATION_TEST_DIR)"'

codegen:
	@$(MAKE) run CMD='-c "./build/codegen.sh"'

DOCS_CMD = "cd docs && make clean &&          \
                doc8 --max-line-length 90 --ignore D000 . && \
                make spelling && make html           \
	   "

docs:
ifeq ($(DOCKER_BUILD),"true")
	@echo "running DOCS_CMD in the containerized build environment"
	@docker run             \
		--entrypoint ''     \
		--rm                \
		-v "$(PWD):/repo"   \
		-w /repo            \
		$(DOCS_BUILD_IMAGE) \
		/bin/bash -c $(DOCS_CMD)
else
	@/bin/bash -c $(DOCS_CMD)
endif

build-dirs:
	@mkdir -p bin/$(ARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(ARCH)

run: build-dirs
ifeq ($(DOCKER_BUILD),"true")
	@echo "running CMD in the containerized build environment"
	@docker run                                                     \
		--rm                                                        \
		--net host                                                  \
		-e GITHUB_TOKEN=$(GITHUB_TOKEN)                             \
		-v "${HOME}/.kube:/root/.kube"                              \
		-v "$(PWD)/.go/pkg:/go/pkg"                                 \
		-v "$(PWD)/.go/cache:/go/.cache"                            \
		-v "$(PWD):/go/src/$(PKG)"                                  \
		-v "$(PWD)/bin/$(ARCH):/go/bin"                             \
		-v "$(PWD)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)" \
		-v "$(DOCKER_CONFIG):/root/.docker"                         \
		-v /var/run/docker.sock:/var/run/docker.sock                \
		-w /go/src/$(PKG)                                           \
		$(BUILD_IMAGE)                                              \
		/bin/sh $(CMD)
else
	@/bin/bash $(CMD)
endif

clean: dotfile-clean bin-clean

dotfile-clean:
	rm -rf .container-* .dockerfile-* .push-* .deploy-*

bin-clean:
	rm -rf .go bin

release-docs: docs
	@if [ -z ${AWS_ACCESS_KEY_ID} ] || [ -z ${AWS_SECRET_ACCESS_KEY} ]; then\
		echo "Please set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY. Exiting.";\
		exit 1;\
	fi;\

	@if [ -f "docs/_build/html/index.html" ]; then\
		aws s3 sync docs/_build/html $(DOCS_RELEASE_BUCKET) --delete;\
		echo "Success";\
	else\
		echo "No built docs found";\
		exit 1;\
	fi;\

release-helm:
	@/bin/bash ./build/release_helm.sh $(VERSION)

gorelease:
	@$(MAKE) run CMD='-c "./build/gorelease.sh"'

release-snapshot:
	@$(MAKE) run CMD='-c "goreleaser --debug release --rm-dist --snapshot"'

go-mod-download:
	@$(MAKE) run CMD='-c "go mod download"'

start-kind:
	@$(MAKE) run CMD='-c "./build/local_kubernetes.sh start_localkube"'

tiller:
	@/bin/bash ./build/init_tiller.sh

stop-kind:
	@$(MAKE) run CMD='-c "./build/local_kubernetes.sh stop_localkube"'

