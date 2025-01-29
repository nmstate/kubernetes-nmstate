SHELL := /bin/bash

PWD = $(shell pwd)
export GO_VERSION = $(shell hack/go-version.sh)

export IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate
NAMESPACE ?= nmstate

ifeq ($(NMSTATE_PIN), future)
HANDLER_EXTRA_PARAMS:= "--build-arg NMSTATE_SOURCE=git"
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

KUBE_RBAC_PROXY_NAME ?= origin-kube-rbac-proxy
KUBE_RBAC_PROXY_TAG ?= 4.10.0
KUBE_RBAC_PROXY_IMAGE_REGISTRY ?= quay.io
KUBE_RBAC_PROXY_IMAGE_REPO ?= openshift
KUBE_RBAC_PROXY_FULL_NAME ?= $(KUBE_RBAC_PROXY_IMAGE_REPO)/$(KUBE_RBAC_PROXY_NAME):$(KUBE_RBAC_PROXY_TAG)
KUBE_RBAC_PROXY_IMAGE ?= $(KUBE_RBAC_PROXY_IMAGE_REGISTRY)/$(KUBE_RBAC_PROXY_FULL_NAME)

export HANDLER_NAMESPACE ?= nmstate
export OPERATOR_NAMESPACE ?= $(HANDLER_NAMESPACE)
export MONITORING_NAMESPACE ?= monitoring
HANDLER_PULL_POLICY ?= Always
OPERATOR_PULL_POLICY ?= Always
export IMAGE_BUILDER ?= $(shell if podman ps >/dev/null 2>&1; then echo podman; elif docker ps >/dev/null 2>&1; then echo docker; fi)

WHAT ?= ./pkg/... ./controllers/...

LINTER_IMAGE_TAG ?= v0.0.3

unit_test_args ?=  -r -keep-going --randomize-all --randomize-suites --race --trace $(UNIT_TEST_ARGS)

export KUBEVIRT_PROVIDER ?= k8s-1.32
export KUBEVIRT_NUM_NODES ?= 3 # 1 control-plane, 2 worker needed for e2e tests
export KUBEVIRT_NUM_SECONDARY_NICS ?= 2

export E2E_TEST_TIMEOUT ?= 80m

e2e_test_args = -vv --show-node-events -timeout=$(E2E_TEST_TIMEOUT) $(E2E_TEST_ARGS)

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
export GOPROXY=direct

export KUBECONFIG ?= $(shell ./cluster/kubeconfig.sh)
export SSH ?= ./cluster/ssh.sh
export KUBECTL ?= ./cluster/kubectl.sh

KUBECTL ?= ./cluster/kubectl.sh
OPERATOR_SDK_VERSION ?= 1.21.0

GINKGO = GOFLAGS=-mod=mod go run github.com/onsi/ginkgo/v2/ginkgo@v2.11.0
CONTROLLER_GEN = GOFLAGS=-mod=mod go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.1
OPM = hack/opm.sh

LOCAL_REGISTRY ?= registry:5000

export MANIFESTS_DIR ?= build/_output/manifests
BUNDLE_DIR ?= ./bundle
BUNDLE_DOCKERFILE ?= bundle.Dockerfile
MANIFEST_BASES_DIR ?= deploy/bases

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

all: check handler operator

check: lint vet whitespace-check gofmt-check promlint-check

format: whitespace-format gofmt

vet:
	GOFLAGS=-mod=mod go vet ./...

whitespace-format:
	hack/whitespace.sh format

gofmt:
	gofmt -l cmd/ test/ hack/ api/ controllers/ pkg/ | grep -v "/vendor/" | xargs -r gofmt -w

whitespace-check:
	hack/whitespace.sh check

gofmt-check:
	test -z "`gofmt -l cmd/ test/ hack/ api/ controllers/ pkg/ | grep -v "/vendor/"`" || (gofmt -l cmd/ test/ hack/ api/ controllers/ pkg/ && exit 1)

promlint-check:
	LINTER_IMAGE_TAG=${LINTER_IMAGE_TAG} hack/prom_metric_linter.sh

lint:
	hack/lint.sh

OPERATOR_SDK = $(CURDIR)/build/_output/bin/operator-sdk_${OPERATOR_SDK_VERSION}
operator-sdk: ## Download operator-sdk locally.
ifneq (,$(shell operator-sdk version 2>/dev/null | grep "operator-sdk version: \"v$(OPERATOR_SDK_VERSION)\"" ))
OPERATOR_SDK = $(shell which operator-sdk)
else
ifeq (,$(wildcard $(OPERATOR_SDK)))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	curl -Lo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/v$(OPERATOR_SDK_VERSION)/operator-sdk_$$(go env GOOS)_$$(go env GOARCH) ;\
	chmod +x $(OPERATOR_SDK) ;\
	}
endif
endif

gen-k8s:
	cd api && $(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

gen-crds:
	cd api && $(CONTROLLER_GEN) crd paths="./..." output:crd:artifacts:config=../deploy/crds

gen-rbac:
	$(CONTROLLER_GEN) crd rbac:roleName=nmstate-operator paths="./controllers/operator/..." output:rbac:artifacts:config=deploy/operator

check-gen: check-manifests check-bundle

check-manifests: generate
	./hack/check-gen.sh generate

check-bundle: bundle
	./hack/check-gen.sh bundle

generate: gen-k8s gen-crds gen-rbac

manifests:
	GOFLAGS=-mod=mod go run hack/render-manifests.go -handler-prefix=$(HANDLER_PREFIX) -handler-namespace=$(HANDLER_NAMESPACE) -operator-namespace=$(OPERATOR_NAMESPACE) -handler-image=$(HANDLER_IMAGE) -operator-image=$(OPERATOR_IMAGE) -handler-pull-policy=$(HANDLER_PULL_POLICY) -monitoring-namespace=$(MONITORING_NAMESPACE) -kube-rbac-proxy-image=$(KUBE_RBAC_PROXY_IMAGE) -operator-pull-policy=$(OPERATOR_PULL_POLICY) -input-dir=deploy/ -output-dir=$(MANIFESTS_DIR)

handler: SKIP_PUSH=true
handler: push-handler

push-handler:
	SKIP_PUSH=$(SKIP_PUSH) SKIP_IMAGE_BUILD=$(SKIP_IMAGE_BUILD) IMAGE=${HANDLER_IMAGE} hack/build-push-container.${IMAGE_BUILDER}.sh ${HANDLER_EXTRA_PARAMS} --build-arg GO_VERSION=$(GO_VERSION) -f build/Dockerfile

operator: SKIP_PUSH=true
operator: push-operator

push-operator:
	SKIP_PUSH=$(SKIP_PUSH) SKIP_IMAGE_BUILD=$(SKIP_IMAGE_BUILD) IMAGE=${OPERATOR_IMAGE} hack/build-push-container.${IMAGE_BUILDER}.sh --build-arg GO_VERSION=$(GO_VERSION) -f build/Dockerfile.operator

push: push-handler push-operator

test/unit/api:
	cd api && $(GINKGO) --junit-report=junit-api-unit-test.xml $(unit_test_args) ./...

test/unit: test/unit/api
	NODE_NAME=node01 $(GINKGO) --junit-report=junit-pkg-controller-unit-test.xml $(unit_test_args) $(WHAT)

test-e2e-handler:
	KUBECONFIG=$(KUBECONFIG) OPERATOR_NAMESPACE=$(OPERATOR_NAMESPACE) MONITORING_NAMESPACE=$(MONITORING_NAMESPACE) $(GINKGO) $(e2e_test_args) ./test/e2e/handler ...

test-e2e-operator: manifests
	KUBECONFIG=$(KUBECONFIG) OPERATOR_NAMESPACE=$(OPERATOR_NAMESPACE) MONITORING_NAMESPACE=$(MONITORING_NAMESPACE) $(GINKGO) $(e2e_test_args) ./test/e2e/operator ...

test-e2e-upgrade: manifests
	./hack/prepare-e2e-test-upgrade.sh
	KUBECONFIG=$(KUBECONFIG) OPERATOR_NAMESPACE=$(OPERATOR_NAMESPACE) MONITORING_NAMESPACE=$(MONITORING_NAMESPACE) $(GINKGO) $(e2e_test_args) ./test/e2e/upgrade ...

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

release-notes:
	hack/render-release-notes.sh $(WHAT) ./api

release:
	hack/release.sh

vendor:
	cd api && go mod tidy
	go mod tidy
	go mod vendor

# Generate bundle manifests and metadata, then validate generated files.
bundle: operator-sdk gen-crds manifests
	cp -r $(MANIFEST_BASES_DIR) $(MANIFESTS_DIR)/bases
	$(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS) --deploy-dir $(MANIFESTS_DIR) --crds-dir deploy/crds
	$(OPERATOR_SDK) bundle validate $(BUNDLE_DIR)

# Build the bundle image.
bundle-build:
	$(IMAGE_BUILDER) build -f $(BUNDLE_DOCKERFILE) -t $(BUNDLE_IMG) .

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
	operator-sdk \
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
