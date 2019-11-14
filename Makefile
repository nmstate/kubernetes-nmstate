IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate

HANDLER_IMAGE_NAME ?= kubernetes-nmstate-handler
HANDLER_IMAGE_SUFFIX ?=
HANDLER_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(HANDLER_IMAGE_NAME)$(HANDLER_IMAGE_SUFFIX)
HANDLER_IMAGE ?= $(IMAGE_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)

unit_test_args ?=  -r --randomizeAllSpecs --randomizeSuites --race --trace $(UNIT_TEST_ARGS)

export KUBEVIRT_PROVIDER ?= k8s-1.14.6
export KUBEVIRT_NUM_NODES ?= 1
export KUBEVIRT_NUM_SECONDARY_NICS ?= 2

export GOFLAGS=-mod=vendor
export GO111MODULE=on

e2e_test_args = -singleNamespace=true -test.v -test.timeout=40m -ginkgo.v -ginkgo.slowSpecThreshold=60 $(E2E_TEST_ARGS)
ifeq ($(findstring k8s,$(KUBEVIRT_PROVIDER)),k8s)
	e2e_test_args += -primaryNic eth0 -firstSecondaryNic eth1 -secondSecondaryNic eth2
else
	e2e_test_args += -primaryNic ens3 -firstSecondaryNic ens8 -secondSecondaryNic ens9
endif

BIN_DIR = build/_output/bin/
GINKGO ?= $(BIN_DIR)/ginkgo
OPERATOR_SDK ?= $(BIN_DIR)/operator-sdk
GITHUB_RELEASE ?= $(BIN_DIR)/github-release
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

check: format vet

format:
	gofmt -d cmd/ pkg/

vet:
	go vet ./cmd/... ./pkg/...

$(GINKGO): go.mod
	GOBIN=$$(pwd)/$(BIN_DIR) go install ./vendor/github.com/onsi/ginkgo/ginkgo

$(OPERATOR_SDK): go.mod
	GOBIN=$$(pwd)/$(BIN_DIR) go install ./vendor/github.com/operator-framework/operator-sdk/cmd/operator-sdk

$(GITHUB_RELEASE): go.mod
	GOBIN=$$(pwd)/$(BIN_DIR) go install ./vendor/github.com/aktau/github-release

handler: $(OPERATOR_SDK)
	$(OPERATOR_SDK) build $(HANDLER_IMAGE)

gen-k8s: $(OPERATOR_SDK)
	$(OPERATOR_SDK) generate k8s

gen-openapi: $(OPERATOR_SDK)
	@echo "WARNING!!!"
	@echo "WARNING!!! kubernets-nmstate has some manual overrides of generated"
	@echo "WARNING!!! openapi code, be sure that code get propertly reviewed by team."
	@echo "WARNING!!!"
	$(OPERATOR_SDK) generate openapi

push-handler: handler
	docker push $(HANDLER_IMAGE)

test/unit: $(GINKGO)
	INTERFACES_FILTER="" NODE_NAME=node01 $(GINKGO) $(unit_test_args) ./pkg/

test/e2e: $(OPERATOR_SDK)
	# We have to unset mod=vendor here since operator-sdk is already
	# building with it, and go tool fail if it's specified twice
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

cluster-up: $(CLUSTER_UP)
	$(CLUSTER_UP)
	hack/install-nm.sh
	hack/flush-secondary-nics.sh
	hack/install-ovs.sh

cluster-down: $(CLUSTER_DOWN)
	$(CLUSTER_DOWN)

cluster-clean: $(KUBECTL)
	$(KUBECTL) delete --ignore-not-found -f build/_output/
	$(KUBECTL) delete --ignore-not-found -f deploy/
	$(KUBECTL) delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkstates_crd.yaml
	$(KUBECTL) delete --ignore-not-found -f deploy/crds/nmstate.io_nodenetworkconfigurationpolicies_crd.yaml
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
	./hack/vendor-tools.sh $$(pwd)/$(BIN_DIR) $$(pwd)/tools.go

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
	release
