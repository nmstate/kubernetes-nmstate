SHELL := /bin/bash

IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate

HANDLER_IMAGE_NAME ?= kubernetes-nmstate-handler
HANDLER_IMAGE_SUFFIX ?=
HANDLER_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(HANDLER_IMAGE_NAME)$(HANDLER_IMAGE_SUFFIX)
HANDLER_IMAGE ?= $(IMAGE_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)

WHAT ?= ./pkg

unit_test_args ?=  -r -keepGoing --randomizeAllSpecs --randomizeSuites --race --trace $(UNIT_TEST_ARGS)

export KUBEVIRT_PROVIDER ?= k8s-1.17.0
export KUBEVIRT_NUM_NODES ?= 1
export KUBEVIRT_NUM_SECONDARY_NICS ?= 2

export E2E_TEST_TIMEOUT ?= 40m

e2e_test_args = -singleNamespace=true -test.v -test.timeout=$(E2E_TEST_TIMEOUT) -ginkgo.v -ginkgo.slowSpecThreshold=60 $(E2E_TEST_ARGS)

ifeq ($(findstring k8s,$(KUBEVIRT_PROVIDER)),k8s)
export PRIMARY_NIC = eth0
export FIRST_SECONDARY_NIC = eth1
export SECOND_SECONDARY_NIC = eth2
else
export PRIMARY_NIC = ens3
export FIRST_SECONDARY_NIC = ens8
export SECOND_SECONDARY_NIC = ens9
endif

BIN_DIR = $(CURDIR)/build/_output/bin/

export GOFLAGS=-mod=vendor
export GO111MODULE=on
export GOROOT=$(BIN_DIR)/go/
export GOBIN=$(GOROOT)/bin/
export PATH := $(GOROOT)/bin:$(PATH)

GINKGO ?= $(GOBIN)/ginkgo
OPERATOR_SDK ?= $(GOBIN)/operator-sdk
GITHUB_RELEASE ?= $(GOBIN)/github-release
GOFMT := $(GOBIN)/gofmt
GO := $(GOBIN)/go

LOCAL_REGISTRY ?= registry:5000

CLUSTER_DIR ?= kubevirtci/cluster-up/
KUBECONFIG ?= kubevirtci/_ci-configs/$(KUBEVIRT_PROVIDER)/.kubeconfig
export KUBECTL ?= $(CLUSTER_DIR)/kubectl.sh
CLUSTER_UP ?= $(CLUSTER_DIR)/up.sh
CLUSTER_DOWN ?= $(CLUSTER_DIR)/down.sh
CLI ?= $(CLUSTER_DIR)/cli.sh
export SSH ?= $(CLUSTER_DIR)/ssh.sh

install_kubevirtci := hack/install-kubevirtci.sh
local_handler_manifest = build/_output/handler.local.yaml
versioned_operator_manifest = build/_output/versioned/operator.yaml
description = build/_output/description

resources = deploy/namespace.yaml deploy/service_account.yaml deploy/role.yaml deploy/role_binding.yaml
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

$(GITHUB_RELEASE): go.mod $(GO)
	$(GO) install ./vendor/github.com/aktau/github-release


gen-k8s: $(OPERATOR_SDK)
	$(OPERATOR_SDK) generate k8s

gen-openapi: $(OPERATOR_SDK)
	$(OPERATOR_SDK) generate openapi

handler: gen-openapi gen-k8s $(OPERATOR_SDK)
	$(OPERATOR_SDK) build $(HANDLER_IMAGE)

push-handler: handler
	docker push $(HANDLER_IMAGE)

test/unit: $(GINKGO)
	INTERFACES_FILTER="" NODE_NAME=node01 $(GINKGO) $(unit_test_args) $(WHAT)

test/e2e: $(OPERATOR_SDK)
	# We have to unset mod=vendor here since operator-sdk is already
	# building with it, and go tool fail if it's specified twice
	mkdir -p test_logs/e2e
	unset GOFLAGS && $(OPERATOR_SDK) test local ./test/e2e \
		--kubeconfig $(KUBECONFIG) \
		--namespace nmstate \
		--no-setup \
		--go-test-flags "$(e2e_test_args)"

$(local_handler_manifest): deploy/operator.yaml
	mkdir -p $(dir $@)
	sed "s#REPLACE_IMAGE#$(LOCAL_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)#" \
		deploy/operator.yaml > $@


$(versioned_operator_manifest): HANDLER_IMAGE_SUFFIX = :$(shell hack/version.sh)
$(versioned_operator_manifest): version/version.go
	mkdir -p $(dir $@)
	sed "s#REPLACE_IMAGE#$(HANDLER_IMAGE)#" \
		deploy/operator.yaml > $@

$(CLUSTER_DIR)/%: $(install_kubevirtci)
	$(install_kubevirtci)

cluster-prepare:
	hack/install-ovs.sh
	hack/install-nm.sh
	hack/flush-secondary-nics.sh

provider-up: $(CLUSTER_UP)
	$(CLUSTER_UP)

cluster-up: provider-up cluster-prepare

cluster-down: $(CLUSTER_DOWN)
	$(CLUSTER_DOWN)

cluster-clean: $(KUBECTL)
	$(KUBECTL) delete --ignore-not-found -f build/_output/
	$(KUBECTL) delete --ignore-not-found -f deploy/
	$(KUBECTL) delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkstates_crd.yaml
	$(KUBECTL) delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
	$(KUBECTL) delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkconfigurationenactments_crd.yaml
	if [[ "$$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$$ ]]; then \
		$(KUBECTL) delete --ignore-not-found -f deploy/openshift/; \
	fi

cluster-sync-resources: $(KUBECTL)
	for resource in $(resources); do \
		$(KUBECTL) apply -f $$resource || exit 1; \
	done
	if [[ "$$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$$ ]]; then \
		$(KUBECTL) apply -f deploy/openshift/; \
	fi

cluster-sync-handler: cluster-sync-resources $(local_handler_manifest)
	if [[ "$$KUBEVIRT_PROVIDER" =~ ^(okd|ocp)-.*$$ ]]; then \
		IMAGE_REGISTRY=localhost:$$($(CLI) ports --container-name=cluster registry | tr -d '\r') \
				   make push-handler;  \
	else \
		IMAGE_REGISTRY=localhost:$$($(CLI) ports registry | tr -d '\r') \
				   make push-handler; \
	fi
	local_handler_manifest=$(local_handler_manifest) ./hack/cluster-sync-handler.sh


cluster-sync: cluster-sync-handler

$(description): version/description
	mkdir -p $(dir $@)
	sed "s#HANDLER_IMAGE#$(HANDLER_IMAGE)#" \
		version/description > $@

prepare-patch:
	./hack/prepare-release.sh patch
prepare-minor:
	./hack/prepare-release.sh minor
prepare-major:
	./hack/prepare-release.sh major

# This uses target specific variables [1] so we can use push-handler as a
# dependency and change the SUFFIX with the correct version so no need for
# calling make on make is needed.
# [1] https://www.gnu.org/software/make/manual/html_node/Target_002dspecific.html
release: HANDLER_IMAGE_SUFFIX = :$(shell hack/version.sh)
release: $(versioned_operator_manifest) push-handler $(description) $(GITHUB_RELEASE)
	DESCRIPTION=$(description) \
	GITHUB_RELEASE=$(GITHUB_RELEASE) \
	TAG=$(shell hack/version.sh) \
				   hack/release.sh \
				   		$(resources) \
						$(versioned_operator_manifest) \
						$(shell find deploy/crds/ deploy/openshift -type f)

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
	cluster-sync-resources \
	cluster-sync-handler \
	cluster-sync \
	cluster-clean \
	release \
	whitespace-check \
	whitespace-format
