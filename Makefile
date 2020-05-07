SHELL := /bin/bash

IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate
NAMESPACE ?= nmstate

HANDLER_IMAGE_NAME ?= kubernetes-nmstate-handler
HANDLER_IMAGE_SUFFIX ?=
HANDLER_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(HANDLER_IMAGE_NAME)$(HANDLER_IMAGE_SUFFIX)
HANDLER_IMAGE ?= $(IMAGE_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)
HANDLER_PREFIX ?=
OPERATOR_IMAGE_NAME ?= kubernetes-nmstate-operator
OPERATOR_IMAGE_SUFFIX ?=
OPERATOR_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(OPERATOR_IMAGE_NAME)$(OPERATOR_IMAGE_SUFFIX)
OPERATOR_IMAGE ?= $(IMAGE_REGISTRY)/$(OPERATOR_IMAGE_FULL_NAME)

export HANDLER_NAMESPACE ?= nmstate
export OPERATOR_NAMESPACE ?= $(HANDLER_NAMESPACE)
HANDLER_PULL_POLICY ?= Always
OPERATOR_PULL_POLICY ?= Always
IMAGE_BUILDER ?= docker

WHAT ?= ./pkg

unit_test_args ?=  -r -keepGoing --randomizeAllSpecs --randomizeSuites --race --trace $(UNIT_TEST_ARGS)

export KUBEVIRT_PROVIDER ?= k8s-1.17
export KUBEVIRT_NUM_NODES ?= 2 # 1 master, 1 worker needed for e2e tests
export KUBEVIRT_NUM_SECONDARY_NICS ?= 2

export E2E_TEST_TIMEOUT ?= 40m

e2e_test_args = -singleNamespace=true -test.v -test.timeout=$(E2E_TEST_TIMEOUT) -ginkgo.v -ginkgo.slowSpecThreshold=60 $(E2E_TEST_ARGS)

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
GITHUB_RELEASE ?= $(GOBIN)/github-release
RELEASE_NOTES ?= $(GOBIN)/release-notes
GOFMT := $(GOBIN)/gofmt
GO := $(GOBIN)/go

LOCAL_REGISTRY ?= registry:5000

export MANIFESTS_DIR ?= build/_output/manifests
description = build/_output/description

all: check handler

check: vet whitespace-check gofmt-check

format: whitespace-format gofmt

vet: $(GO)
	$(GO) vet ./cmd/... ./pkg/... ./test/...

whitespace-format:
	hack/whitespace.sh format

gofmt: $(GO)
	$(GOFMT) -w cmd/ pkg/ test/e2e/

whitespace-check:
	hack/whitespace.sh check

gofmt-check: $(GO)
	test -z "`$(GOFMT) -l cmd/ pkg/ test/e2e/`" || ($(GOFMT) -l cmd/ pkg/ test/e2e/ && exit 1)

$(GO):
	hack/install-go.sh $(BIN_DIR)

$(GINKGO): go.mod $(GO)
	$(GO) install ./vendor/github.com/onsi/ginkgo/ginkgo

$(OPERATOR_SDK): go.mod $(GO)
	$(GO) install ./vendor/github.com/operator-framework/operator-sdk/cmd/operator-sdk

$(OPENAPI_GEN): go.mod $(GO)
	$(GO) install ./vendor/k8s.io/kube-openapi/cmd/openapi-gen

$(GITHUB_RELEASE): go.mod $(GO)
	$(GO) install ./vendor/github.com/aktau/github-release

$(RELEASE_NOTES): go.mod $(GO)
	$(GO) install ./vendor/k8s.io/release/cmd/release-notes

gen-k8s: $(OPERATOR_SDK)
	$(OPERATOR_SDK) generate k8s

gen-openapi: $(OPENAPI_GEN)
	$(OPENAPI_GEN) --logtostderr=true -o "" -i ./pkg/apis/nmstate/v1alpha1 -O zz_generated.openapi -p ./pkg/apis/nmstate/v1alpha1 -h ./hack/boilerplate.go.txt -r "-"
	$(OPENAPI_GEN) --logtostderr=true -o "" -i ./pkg/apis/nmstate/v1beta1 -O zz_generated.openapi -p ./pkg/apis/nmstate/v1beta1 -h ./hack/boilerplate.go.txt -r "-"

gen-crds: $(OPERATOR_SDK)
	$(OPERATOR_SDK) generate crds

manifests: $(GO)
	$(GO) run hack/render-manifests.go -handler-prefix=$(HANDLER_PREFIX) -handler-namespace=$(HANDLER_NAMESPACE) -operator-namespace=$(OPERATOR_NAMESPACE) -handler-image=$(HANDLER_IMAGE) -operator-image=$(OPERATOR_IMAGE) -handler-pull-policy=$(HANDLER_PULL_POLICY) -operator-pull-policy=$(OPERATOR_PULL_POLICY) -input-dir=deploy/ -output-dir=$(MANIFESTS_DIR)

handler: gen-openapi gen-k8s gen-crds $(OPERATOR_SDK)
	$(OPERATOR_SDK) build $(HANDLER_IMAGE) --image-builder $(IMAGE_BUILDER)
push-handler: handler
	$(IMAGE_BUILDER) push $(HANDLER_IMAGE)
operator: handler
	$(IMAGE_BUILDER) build --build-arg=BASE_IMAGE=$(HANDLER_IMAGE) -t $(OPERATOR_IMAGE) -f build/Dockerfile.operator .
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

$(description): version/description
	mkdir -p $(dir $@)
	cp version/description > $@

prepare-patch: $(RELEASE_NOTES)
	RELEASE_NOTES=$(RELEASE_NOTES) ./hack/prepare-release.sh patch
prepare-minor: $(RELEASE_NOTES)
	RELEASE_NOTES=$(RELEASE_NOTES) ./hack/prepare-release.sh minor
prepare-major: $(RELEASE_NOTES)
	RELEASE_NOTES=$(RELEASE_NOTES) ./hack/prepare-release.sh major

# This uses target specific variables [1] so we can use push-handler as a
# dependency and change the SUFFIX with the correct version so no need for
# calling make on make is needed.
# [1] https://www.gnu.org/software/make/manual/html_node/Target_002dspecific.html
release: OPERATOR_IMAGE_SUFFIX =: $(shell hack/version.sh)
release: HANDLER_IMAGE_SUFFIX =: $(shell hack/version.sh)
release: manifests push-handler push-operator $(description) $(GITHUB_RELEASE) version/version.go
	DESCRIPTION=$(description) \
	GITHUB_RELEASE=$(GITHUB_RELEASE) \
	TAG=$(shell hack/version.sh) \
				   hack/release.sh \
						$(shell find $(MANIFESTS_DIR) -type f) \
						deploy/crds/nmstate.io_nmstates_crd.yaml \
						deploy/crds/nmstate.io_v1alpha1_nmstate_cr.yaml

vendor: $(GO)
	$(GO) mod tidy
	$(GO) mod vendor

tools-vendoring:
	./hack/vendor-tools.sh $(BIN_DIR) $$(pwd)/tools.go

.PHONY: \
	all \
	check \
	format \
	vet \
	handler \
	push-handler \
	test/unit \
	test/e2e \
	cluster-up \
	cluster-down \
	cluster-sync-handler \
	cluster-sync \
	cluster-clean \
	release \
	vendor \
	whitespace-check \
	whitespace-format \
	manifests
