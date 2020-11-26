SHELL := /bin/bash

IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate
NAMESPACE ?= nmstate

HANDLER_IMAGE_NAME ?= kubernetes-nmstate-handler
HANDLER_IMAGE_TAG ?= latest
HANDLER_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(HANDLER_IMAGE_NAME):$(HANDLER_IMAGE_TAG)
HANDLER_IMAGE ?= $(IMAGE_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)
HANDLER_PREFIX ?=
OPERATOR_IMAGE_NAME ?= kubernetes-nmstate-operator
OPERATOR_IMAGE_TAG ?= latest
OPERATOR_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(OPERATOR_IMAGE_NAME):$(OPERATOR_IMAGE_TAG)
OPERATOR_IMAGE ?= $(IMAGE_REGISTRY)/$(OPERATOR_IMAGE_FULL_NAME)

export HANDLER_NAMESPACE ?= nmstate
export OPERATOR_NAMESPACE ?= $(HANDLER_NAMESPACE)
HANDLER_PULL_POLICY ?= Always
OPERATOR_PULL_POLICY ?= Always
IMAGE_BUILDER ?= docker

WHAT ?= ./pkg ./controllers ./api

unit_test_args ?=  -r -keepGoing --randomizeAllSpecs --randomizeSuites --race --trace $(UNIT_TEST_ARGS)

export KUBEVIRT_PROVIDER ?= k8s-1.19
export KUBEVIRT_NUM_NODES ?= 2 # 1 master, 1 worker needed for e2e tests
export KUBEVIRT_NUM_SECONDARY_NICS ?= 2

export E2E_TEST_TIMEOUT ?= 60m

e2e_test_args = -v -timeout=$(E2E_TEST_TIMEOUT) -slowSpecThreshold=60 $(E2E_TEST_ARGS)

ifeq ($(findstring k8s,$(KUBEVIRT_PROVIDER)),k8s)
export PRIMARY_NIC ?= eth0
export FIRST_SECONDARY_NIC ?= eth1
export SECOND_SECONDARY_NIC ?= eth2
else
export PRIMARY_NIC ?= ens3
export FIRST_SECONDARY_NIC ?= ens8
export SECOND_SECONDARY_NIC ?= ens9
endif
BIN_DIR = $(CURDIR)/build/_output/bin/

export GOPROXY=direct
export GOSUMDB=off
export GOFLAGS=-mod=vendor
export GOROOT=$(BIN_DIR)/go/
export GOBIN=$(GOROOT)/bin/
export PATH := $(GOROOT)/bin:$(PATH)

export KUBECONFIG ?= $(shell ./cluster/kubeconfig.sh)
export SSH ?= ./cluster/ssh.sh
export KUBECTL ?= ./cluster/kubectl.sh

KUBECTL ?= ./cluster/kubectl.sh
GINKGO ?= $(GOBIN)/ginkgo
CONTROLLER_GEN ?= $(GOBIN)/controller-gen
export GITHUB_RELEASE ?= $(GOBIN)/github-release
export RELEASE_NOTES ?= $(GOBIN)/release-notes
GOFMT := $(GOBIN)/gofmt
export GO := $(GOBIN)/go
OPM ?= $(GOBIN)/opm
OPERATOR_SDK ?= $(GOBIN)/operator-sdk

LOCAL_REGISTRY ?= registry:5000

export MANIFESTS_DIR ?= build/_output/manifests

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?= $(IMAGE_REGISTRY)/$(IMAGE_REPO)/kubernetes-nmstate-operator-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

INDEX_VERSION ?= 1.0.0
# Default index image tag
INDEX_IMG ?= $(IMAGE_REGISTRY)/$(IMAGE_REPO)/kubernetes-nmstate-operator-index:$(INDEX_VERSION)

all: check handler

check: vet whitespace-check gofmt-check

format: whitespace-format gofmt

vet: $(GO)
	$(GO) vet ./...

whitespace-format:
	hack/whitespace.sh format

gofmt: $(GO)
	$(GOFMT) -w *.go test/ hack/ api/ controllers/ pkg/

whitespace-check:
	hack/whitespace.sh check

gofmt-check: $(GO)
	test -z "`$(GOFMT) -l *.go test/ hack/ api/ controllers/ pkg/`" || ($(GOFMT) -l *.go test/ hack/ api/ controllers/ pkg/ && exit 1)

$(GO):
	hack/install-go.sh $(BIN_DIR)

$(GINKGO): go.mod tools
$(OPENAPI_GEN): go.mod tools
$(GITHUB_RELEASE): go.mod tools
$(RELEASE_NOTES): go.mod tools
$(CONTROLLER_GEN): go.mod tools
$(OPM): go.mod tools
$(OPERATOR_SDK): go.mod tools

gen-k8s: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

gen-crds: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=deploy/crds

gen-rbac: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=nmstate-operator paths="./controllers/nmstate_controller.go" output:rbac:artifacts:config=deploy/operator

check-gen: generate
	./hack/check-gen.sh

generate: gen-k8s gen-crds gen-rbac

manifests: $(GO)
	$(GO) run hack/render-manifests.go -handler-prefix=$(HANDLER_PREFIX) -handler-namespace=$(HANDLER_NAMESPACE) -operator-namespace=$(OPERATOR_NAMESPACE) -handler-image=$(HANDLER_IMAGE) -operator-image=$(OPERATOR_IMAGE) -handler-pull-policy=$(HANDLER_PULL_POLICY) -operator-pull-policy=$(OPERATOR_PULL_POLICY) -input-dir=deploy/ -output-dir=$(MANIFESTS_DIR)

manager: $(GO)
	$(GO) build -o $(BIN_DIR)/manager main.go

handler: manager
	$(IMAGE_BUILDER) build . -f build/Dockerfile -t ${HANDLER_IMAGE}

push-handler: handler
	$(IMAGE_BUILDER) push $(HANDLER_IMAGE)

operator: manager
	$(IMAGE_BUILDER) build . -f build/Dockerfile.operator -t $(OPERATOR_IMAGE)

push-operator: operator
	$(IMAGE_BUILDER) push $(OPERATOR_IMAGE)
push: push-handler push-operator

test/unit: $(GINKGO)
	INTERFACES_FILTER="" NODE_NAME=node01 $(GINKGO) $(unit_test_args) $(WHAT)

test-e2e-handler: $(GINKGO)
	KUBECONFIG=$(shell ./cluster/kubeconfig.sh) $(GINKGO) $(e2e_test_args) ./test/e2e/handler ... -- $(E2E_TEST_SUITE_ARGS)

test-e2e-operator: manifests $(GINKGO)
	KUBECONFIG=$(shell ./cluster/kubeconfig.sh) $(GINKGO) $(e2e_test_args) ./test/e2e/operator ... -- $(E2E_TEST_SUITE_ARGS)

test-e2e: test-e2e-operator test-e2e-handler

cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-clean:
	./cluster/clean.sh

cluster-sync:
	./cluster/sync.sh

cluster-sync-operator:
	./cluster/sync-operator.sh

version-patch:
	./hack/tag-version.sh patch
version-minor:
	./hack/tag-version.sh minor
version-major:
	./hack/tag-version.sh major

release: $(GITHUB_RELEASE) $(RELEASE_NOTES)
	hack/release.sh

vendor: $(GO)
	$(GO) mod tidy
	$(GO) mod vendor

tools: $(GO)
	./hack/install-tools.sh

# Generate bundle manifests and metadata, then validate generated files.
bundle: $(OPERATOR_SDK) gen-crds manifests
	$(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS) --deploy-dir $(MANIFESTS_DIR) --crds-dir deploy/crds
	$(OPERATOR_SDK) bundle validate ./bundle

# Build the bundle image.
bundle-build:
	$(IMAGE_BUILDER) build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# Build the index
index-build: $(OPM) bundle-build
	$(OPM) index add --bundles $(BUNDLE_IMG) --tag $(INDEX_IMG)

bundle-push: bundle-build
	$(IMAGE_BUILDER) push $(BUNDLE_IMG)

index-push: index-build
	$(IMAGE_BUILDER) push $(INDEX_IMG)

olm-push: bundle-push index-push

.PHONY: \
	all \
	check \
	format \
	vet \
	handler \
	push-handler \
	test/unit \
	generate \
	check-gen \
	test-e2e-handler \
	test-e2e-operator \
	test-e2e \
	cluster-up \
	cluster-down \
	cluster-sync-operator \
	cluster-sync \
	cluster-clean \
	release \
	vendor \
	whitespace-check \
	whitespace-format \
	generate-manifests \
	tools \
	bundle \
	bundle-build
