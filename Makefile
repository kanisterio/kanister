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

# Mention the vm-driver that should be used to install OpenShift
vm-driver ?= "kvm"

# Refers to https://github.com/kopia/kopia/commit/317cc36892707ab9bdc5f6e4dea567d1e638a070
KOPIA_COMMIT_ID ?= "317cc36"

KOPIA_REPO ?= "kopia"
# Default OCP version in which the OpenShift apps are going to run
ocp_version ?= "4.10"
###
### These variables should not need tweaking.
###

SRC_DIRS := cmd pkg # directories which hold app source (not vendored)

ALL_ARCH := amd64 arm arm64 ppc64le

# Set default base image dynamically for each arch

IMAGE_NAME := $(BIN)

IMAGE := $(REGISTRY)/$(IMAGE_NAME)

BUILD_IMAGE ?= ghcr.io/kanisterio/build:v0.0.22

# tag 0.1.0 is, 0.0.1 (latest) + gh + aws + helm binary
DOCS_BUILD_IMAGE ?= ghcr.io/kanisterio/docker-sphinx:0.2.0

DOCS_RELEASE_BUCKET ?= s3://docs.kanister.io

GITHUB_TOKEN ?= ""

GOBORING ?= ""

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
	@$(MAKE) run CMD='-c " \
	goreleaser build --id $(BIN) --rm-dist --debug --snapshot \
	&& cp dist/$(BIN)_linux_$(ARCH)/$(BIN) bin/$(ARCH)/$(BIN) \
	"'

bin/$(ARCH)/$(BIN):
	@echo "building: $@"
	@$(MAKE) run CMD='-c " \
		GOARCH=$(ARCH)       \
		VERSION=$(VERSION) \
		PKG=$(PKG)         \
		BIN=$(BIN) \
		GOBORING=$(GOBORING) \
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

helm-test: build-dirs
	@$(MAKE) run CMD='-c "./build/helm-test.sh $(SRC_DIRS)"'

integration-test: build-dirs
	@$(MAKE) run CMD='-c "./build/integration-test.sh short"'

openshift-test:
	@/bin/bash ./build/integration-test.sh openshift $(ocp_version)

golint:
	@$(MAKE) run CMD='-c "./build/golint.sh"'

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
		/bin/bash $(CMD)
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
	@$(MAKE) run CMD='-c "GORELEASER_CURRENT_TAG=v9.99.9-dev goreleaser --debug release --rm-dist --snapshot"'

update-kopia-image:
	@/bin/bash ./build/update_kopia_image.sh -c $(KOPIA_COMMIT_ID) -r $(KOPIA_REPO) -b $(GOBORING)

go-mod-download:
	@$(MAKE) run CMD='-c "go mod download"'

start-kind:
	@$(MAKE) run CMD='-c "./build/local_kubernetes.sh start_localkube"'

tiller:
	@/bin/bash ./build/init_tiller.sh

install-minio:
	@$(MAKE) run CMD='-c "./build/minio.sh install_minio"'

install-csi-hostpath-driver:
	@$(MAKE) run CMD='-c "./build/local_kubernetes.sh install_csi_hostpath_driver"'

uninstall-minio:
	@$(MAKE) run CMD='-c "./build/minio.sh uninstall_minio"'

start-minishift:
	@/bin/bash ./build/minishift.sh start_minishift $(vm-driver)

stop-minishift:
	@/bin/bash ./build/minishift.sh stop_minishift

stop-kind:
	@$(MAKE) run CMD='-c "./build/local_kubernetes.sh stop_localkube"'

check:
	@./build/check.sh

gomod:
	@$(MAKE) run CMD='-c "./build/gomod.sh"'

####################################################################################################################################
# KUBEBUILDER TARGETS
####################################################################################################################################

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.25.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1" output:crd:artifacts:config=pkg/customresource

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="pkg/hack/boilerplate.go.txt" paths="github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: kubebuilder-test
kubebuilder-test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: kubebuilder-build
kubebuilder-build: generate fmt vet ## Build manager binary.
	go build -o bin/manager ./cmd/reposervercontroller/main.go

.PHONY: kubebuilder-run
kubebuilder-run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/reposervercontroller/main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: kubebuilder-test ## Build docker image with the manager.
	docker build -t ${IMG} docker/repositoryserver-controller

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have enable BuildKit, More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> than the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: kubebuilder-test ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' docker/repositoryserver-controller/Dockerfile > docker/repositoryserver-controller/Dockerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f docker/repositoryserver-controller/Dockerfile.cross
	- docker buildx rm project-v3-builder
	rm docker/repositoryserver-controller/Dockerfile.cross

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build pkg/customresource | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build pkg/customresource | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: kubebuilder-deploy
kubebuilder-deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: kubebuilder-undeploy
kubebuilder-undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.9.2

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

####################################################################################################################################
