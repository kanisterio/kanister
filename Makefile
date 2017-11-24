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
DOCKER_BUILD="true"

###
### These variables should not need tweaking.
###

SRC_DIRS := cmd pkg # directories which hold app source (not vendored)

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

#TODO(tom) We can consider open sourcing this repo.
BUILD_IMAGE ?= kanisterio/build:0.13.1-go1.9

DEFAULT_PATH := /usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

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

bin/$(ARCH)/$(BIN): .vendor
	@echo "building: $@"
	@$(MAKE) run CMD='-c " \
		ARCH=$(ARCH)       \
		VERSION=$(VERSION) \
		PKG=$(PKG)         \
		./build/build.sh   \
	"'
# Example: make shell CMD="-c 'date > datefile'"
shell: build-dirs
	@echo "launching a shell in the containerized build environment"
	@docker run                                                            \
	    -ti                                                                \
	    --rm                                                               \
	    -e GOPATH=/go                                                      \
	    -e GOROOT=/usr/local/go                                            \
	    -v "$(PWD)/.go:/go"                                                \
	    -v "$(PWD):/go/src/$(PKG)"                                         \
	    -v "$(PWD)/bin/$(ARCH):/go/bin"                                    \
	    -v "$(PWD)/bin/$(ARCH):/go/bin/$$(go env GOOS)_$(ARCH)"            \
	    -v "$(PWD)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static" \
	    -w /go/src/$(PKG)                                                  \
	    $(BUILD_IMAGE)                                                     \
		ash -l

.vendor:
	@$(MAKE) run CMD='-c "              \
		PATH=$$GOPATH/bin:$$GOROOT/bin::$${PATH} \
		glide install --strip-vendor   \
	"'
	@touch $@

DOTFILE_IMAGE = $(subst :,_,$(subst /,_,$(IMAGE))-$(VERSION))

container: .container-$(DOTFILE_IMAGE) container-name
.container-$(DOTFILE_IMAGE): bin/$(ARCH)/$(BIN) Dockerfile.in
	@ash -c "              \
		BIN=$(BIN)               \
		ARCH=$(ARCH)             \
		BASEIMAGE=$(BASEIMAGE)   \
		IMAGE=$(IMAGE)           \
		VERSION=$(VERSION)       \
		SOURCE_BIN=bin/$(ARCH)/$(BIN) \
		./build/package.sh       \
	"

container-name:
	@echo "container: $(IMAGE):$(VERSION)"

push: .push-$(DOTFILE_IMAGE) push-name
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

.PHONY: test build-dirs run clean container-clean bin-clean vendor-clean

deploy: push .deploy-$(DOTFILE_IMAGE)
.deploy-$(DOTFILE_IMAGE):
	@sed                        \
	    -e 's|IMAGE|$(IMAGE)|g' \
	    -e 's|TAG|$(VERSION)|g' \
	    bundle.yaml.in > .deploy-$(DOTFILE_IMAGE)
	@$(MAKE) run CMD='-c "kubectl apply -f .deploy-$(DOTFILE_IMAGE)"'

test: .vendor build-dirs
	@$(MAKE) run CMD='-c "./build/test.sh $(SRC_DIRS)"'

build-dirs:
	@mkdir -p bin/$(ARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(ARCH)

run: build-dirs
ifeq ($(DOCKER_BUILD),"true")
	@echo "running CMD in the containerized build environment"
	@docker run                                                            \
		--rm                                                               \
		-e GOPATH=/go                                                      \
		-e GOROOT=/usr/local/go                                            \
		-v "${HOME}/.kube:/root/.kube"                                     \
		-v "$(PWD)/.go:/go"                                                \
		-v "$(PWD):/go/src/$(PKG)"                                         \
		-v "$(PWD)/bin/$(ARCH):/go/bin"                                    \
		-v "$(PWD)/bin/$(ARCH):/go/bin/$$(go env GOOS)_$(ARCH)"            \
		-v "$(PWD)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static" \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		ash $(CMD)
else
	@ash $(CMD)
endif

clean: dotfile-clean bin-clean vendor-clean

dotfile-clean:
	rm -rf .container-* .dockerfile-* .push-* .vendor .deploy-*

bin-clean:
	rm -rf .go bin

vendor-clean:
	rm -rf vendor
