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

export E2E_TEST_TIMEOUT ?= 40m

e2e_test_args = -test.v -test.timeout=$(E2E_TEST_TIMEOUT) -ginkgo.v -ginkgo.slowSpecThreshold=60 $(E2E_TEST_ARGS)

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

GINKGO ?= $(GOBIN)/ginkgo
OPERATOR_SDK ?= $(GOBIN)/operator-sdk
OPENAPI_GEN ?= $(GOBIN)/openapi-gen
CONTROLLER_GEN ?= $(GOBIN)/controller-gen
export GITHUB_RELEASE ?= $(GOBIN)/github-release
export RELEASE_NOTES ?= $(GOBIN)/release-notes
GOFMT := $(GOBIN)/gofmt
export GO := $(GOBIN)/go

LOCAL_REGISTRY ?= registry:5000

export MANIFESTS_DIR ?= build/_output/manifests

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

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

$(GINKGO): go.mod
	$(MAKE) tools
$(OPERATOR_SDK): go.mod
	$(MAKE) tools
$(OPENAPI_GEN): go.mod
	$(MAKE) tools
$(GITHUB_RELEASE): go.mod
	$(MAKE) tools
$(RELEASE_NOTES): go.mod
	$(MAKE) tools
$(CONTROLLER_GEN): go.mod
	$(MAKE) tools

gen-k8s: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

gen-openapi: $(OPENAPI_GEN)
	$(OPENAPI_GEN) --logtostderr=true -o "" -i ./api/v1alpha1 -O zz_generated.openapi -p ./api/v1alpha1 -h ./hack/boilerplate.go.txt -r "-"
	$(OPENAPI_GEN) --logtostderr=true -o "" -i ./api/v1beta1 -O zz_generated.openapi -p ./api/v1beta1 -h ./hack/boilerplate.go.txt -r "-"
	$(OPENAPI_GEN) --logtostderr=true -o "" -i ./api/shared -O zz_generated.openapi -p ./api/shared -h ./hack/boilerplate.go.txt -r "-"

gen-crds: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=deploy/crds

check-gen: generate
	./hack/check-gen.sh

generate: gen-openapi gen-k8s gen-crds

manifests: $(GO)
	$(GO) run hack/render-manifests.go -handler-prefix=$(HANDLER_PREFIX) -handler-namespace=$(HANDLER_NAMESPACE) -operator-namespace=$(OPERATOR_NAMESPACE) -handler-image=$(HANDLER_IMAGE) -operator-image=$(OPERATOR_IMAGE) -handler-pull-policy=$(HANDLER_PULL_POLICY) -operator-pull-policy=$(OPERATOR_PULL_POLICY) -input-dir=deploy/ -output-dir=$(MANIFESTS_DIR)

manager: $(OPERATOR_SDK)
	$(GO) build -o $(BIN_DIR)/manager main.go

handler: manager
	$(IMAGE_BUILDER) build . -f build/Dockerfile -t ${HANDLER_IMAGE}

push-handler: handler
	$(IMAGE_BUILDER) push $(HANDLER_IMAGE)

operator: handler
	$(IMAGE_BUILDER) build . -f build/Dockerfile.operator -t $(OPERATOR_IMAGE)

push-operator: operator
	$(IMAGE_BUILDER) push $(OPERATOR_IMAGE)
push: push-handler push-operator

test/unit: $(GINKGO)
	INTERFACES_FILTER="" NODE_NAME=node01 $(GINKGO) $(unit_test_args) $(WHAT)

test-e2e-handler: $(OPERATOR_SDK)
	OPERATOR_SDK="$(OPERATOR_SDK)" TEST_ARGS="$(e2e_test_args)" ./hack/run-e2e-test-handler.sh

test-e2e-operator: manifests $(OPERATOR_SDK)
	OPERATOR_SDK="$(OPERATOR_SDK)" TEST_ARGS="$(e2e_test_args)" KUBECTL=$(KUBECTL) MANIFESTS_DIR=$(MANIFESTS_DIR) ./hack/run-e2e-test-operator.sh

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
	manifests \
	tools
