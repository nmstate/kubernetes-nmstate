SHELL := /bin/bash

PWD = $(shell pwd)
GO_VERSION = $(shell hack/go-version.sh)

export IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate
NAMESPACE ?= nmstate

ifeq ($(NMSTATE_PIN), future)
HANDLER_EXTRA_PARAMS:= "--build-arg NMSTATE_SOURCE=git --build-arg FROM=quay.io/centos/centos:stream9"
endif

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
export IMAGE_BUILDER ?= docker

WHAT ?= ./pkg/... ./controllers/...

unit_test_args ?=  -r -keep-going --randomize-all --randomize-suites --race --trace $(UNIT_TEST_ARGS)

export KUBEVIRT_PROVIDER ?= k8s-1.23
export KUBEVIRT_NUM_NODES ?= 2 # 1 control-plane, 1 worker needed for e2e tests
export KUBEVIRT_NUM_SECONDARY_NICS ?= 2

export E2E_TEST_TIMEOUT ?= 80m

e2e_test_args = -v -timeout=$(E2E_TEST_TIMEOUT) --slow-spec-threshold=60s $(E2E_TEST_ARGS)

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

export GOFLAGS=-mod=vendor

export KUBECONFIG ?= $(shell ./cluster/kubeconfig.sh)
export SSH ?= ./cluster/ssh.sh
export KUBECTL ?= ./cluster/kubectl.sh

KUBECTL ?= ./cluster/kubectl.sh
OPERATOR_SDK ?= $(GOBIN)/operator-sdk

GINKGO = go run github.com/onsi/ginkgo/v2/ginkgo
CONTROLLER_GEN = go run sigs.k8s.io/controller-tools/cmd/controller-gen
OPM = go run -tags=json1 github.com/operator-framework/operator-registry/cmd/opm

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

SKIP_IMAGE_BUILD ?= false

all: check handler

check: lint vet whitespace-check gofmt-check

format: whitespace-format gofmt

vet:
	go vet ./...

whitespace-format:
	hack/whitespace.sh format

gofmt:
	gofmt -l cmd/ test/ hack/ api/ controllers/ pkg/ | grep -v "/vendor/" | xargs -r gofmt -w

whitespace-check:
	hack/whitespace.sh check

gofmt-check:
	test -z "`gofmt -l cmd/ test/ hack/ api/ controllers/ pkg/ | grep -v "/vendor/"`" || (gofmt -l cmd/ test/ hack/ api/ controllers/ pkg/ && exit 1)

lint:
	hack/lint.sh

$(OPERATOR_SDK):
	curl https://github.com/operator-framework/operator-sdk/releases/download/v1.15.0/operator-sdk_linux_amd64 -o $(OPERATOR_SDK)

gen-k8s:
	$(MAKE) -C api gen-k8s

gen-crds:
	$(MAKE) -C api gen-crds

gen-rbac:
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=nmstate-operator paths="./controllers/operator/nmstate_controller.go" output:rbac:artifacts:config=deploy/operator

check-gen: generate
	./hack/check-gen.sh

generate: gen-k8s gen-crds gen-rbac

manifests:
	go run hack/render-manifests.go -handler-prefix=$(HANDLER_PREFIX) -handler-namespace=$(HANDLER_NAMESPACE) -operator-namespace=$(OPERATOR_NAMESPACE) -handler-image=$(HANDLER_IMAGE) -operator-image=$(OPERATOR_IMAGE) -handler-pull-policy=$(HANDLER_PULL_POLICY) -operator-pull-policy=$(OPERATOR_PULL_POLICY) -input-dir=deploy/ -output-dir=$(MANIFESTS_DIR)

handler: SKIP_PUSH=true
handler: push-handler

push-handler:
	SKIP_PUSH=$(SKIP_PUSH) SKIP_IMAGE_BUILD=$(SKIP_IMAGE_BUILD) IMAGE=${HANDLER_IMAGE} hack/build-push-container.${IMAGE_BUILDER}.sh ${HANDLER_EXTRA_PARAMS} . -f build/Dockerfile

operator: SKIP_PUSH=true
operator: push-operator

push-operator:
	SKIP_PUSH=$(SKIP_PUSH) SKIP_IMAGE_BUILD=$(SKIP_IMAGE_BUILD) IMAGE=${OPERATOR_IMAGE} hack/build-push-container.${IMAGE_BUILDER}.sh  . -f build/Dockerfile.operator

push: push-handler push-operator

test/unit/api:
	cd api && $(GINKGO) --junit-report=junit-api-unit-test.xml $(unit_test_args) ./...

test/unit: test/unit/api
	NODE_NAME=node01 $(GINKGO) --junit-report=junit-pkg-controller-unit-test.xml $(unit_test_args) $(WHAT)

test-e2e-handler:
	KUBECONFIG=$(KUBECONFIG) OPERATOR_NAMESPACE=$(OPERATOR_NAMESPACE) $(GINKGO) $(e2e_test_args) ./test/e2e/handler ...

test-e2e-operator: manifests
	KUBECONFIG=$(KUBECONFIG) OPERATOR_NAMESPACE=$(OPERATOR_NAMESPACE) $(GINKGO) $(e2e_test_args) ./test/e2e/operator ...

test-e2e-upgrade: manifests
	KUBECONFIG=$(KUBECONFIG) OPERATOR_NAMESPACE=$(OPERATOR_NAMESPACE) GINKGO="$(GINKGO)" ./hack/run-e2e-test-upgrade.sh $(e2e_test_args) $(E2E_TEST_SUITE_ARGS)

test-e2e: test-e2e-operator test-e2e-handler

test-e2e-ocp: 
	./hack/ocp-e2e-tests.sh

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

release-notes:
	hack/render-release-notes.sh $(WHAT) ./api

release:
	hack/release.sh

vendor-api:
	cd api && go mod tidy -compat=$(GO_VERSION) && go mod vendor

vendor: vendor-api
	go mod tidy -compat=$(GO_VERSION)
	go mod vendor

# Generate bundle manifests and metadata, then validate generated files.
bundle: $(OPERATOR_SDK) gen-crds manifests
	cp -r deploy/bases $(MANIFESTS_DIR)/bases
	$(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS) --deploy-dir $(MANIFESTS_DIR) --crds-dir deploy/crds
	$(OPERATOR_SDK) bundle validate ./bundle

# Build the bundle image.
bundle-build:
	$(IMAGE_BUILDER) build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# Build the index
index-build: bundle-build
	$(OPM) index add --bundles $(BUNDLE_IMG) --tag $(INDEX_IMG) --build-tool $(IMAGE_BUILDER)

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
	bundle-build \
	manifests
