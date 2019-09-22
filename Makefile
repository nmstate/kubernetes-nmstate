IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate

HANDLER_IMAGE_NAME ?= kubernetes-nmstate-handler
HANDLER_IMAGE_SUFFIX ?=
HANDLER_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(HANDLER_IMAGE_NAME)$(HANDLER_IMAGE_SUFFIX)
IMAGE_TAG ?= "latest"
HANDLER_IMAGE ?= $(IMAGE_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME):$(IMAGE_TAG)

UNIT_TEST_ARGS ?=  -r --randomizeAllSpecs --randomizeSuites --race --trace $(UNIT_TEST_EXTRA_ARGS)
ifdef UNIT_TEST_FOCUS
	UNIT_TEST_ARGS += --focus $(UNIT_TEST_FOCUS)
endif
ifdef UNIT_TEST_SKIP
	UNIT_TEST_ARGS += --skip $(UNIT_TEST_SKIP)
endif
ifdef UNIT_TEST_EXTRA_ARGS
	UNIT_TEST_ARGS += $(UNIT_TEST_ARGS)
endif

E2E_TEST_ARGS ?= -test.v
ifdef E2E_TEST_FOCUS
	E2E_TEST_ARGS +=  -ginkgo.focus $(E2E_TEST_FOCUS)
endif
ifdef E2E_TEST_SKIP
	E2E_TEST_ARGS +=  -ginkgo.skip $(E2E_TEST_SKIP)
endif
ifdef E2E_TEST_EXTRA_ARGS
	E2E_TEST_ARGS +=  $(E2E_TEST_EXTRA_ARGS)
endif

GINKGO ?= build/_output/bin/ginkgo
OPERATOR_SDK ?= build/_output/bin/operator-sdk
GITHUB_RELEASE ?= build/_output/bin/github-release
LOCAL_REGISTRY ?= registry:5000

export KUBEVIRT_PROVIDER ?= k8s-1.13.3
export KUBEVIRT_NUM_NODES ?= 1
export KUBEVIRT_NUM_SECONDARY_NICS ?= 2

CLUSTER_DIR ?= kubevirtci/cluster-up/
KUBECONFIG ?= kubevirtci/_ci-configs/$(KUBEVIRT_PROVIDER)/.kubeconfig
KUBECTL ?= $(CLUSTER_DIR)/kubectl.sh
CLUSTER_UP ?= $(CLUSTER_DIR)/up.sh
CLUSTER_DOWN ?= $(CLUSTER_DIR)/down.sh
CLI ?= $(CLUSTER_DIR)/cli.sh
SSH ?= $(CLUSTER_DIR)/ssh.sh

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

$(GINKGO): Gopkg.toml
	GOBIN=$$(pwd)/build/_output/bin/ go install ./vendor/github.com/onsi/ginkgo/ginkgo

$(OPERATOR_SDK): Gopkg.toml
	GOBIN=$$(pwd)/build/_output/bin/ go install ./vendor/github.com/operator-framework/operator-sdk/cmd/operator-sdk

$(GITHUB_RELEASE): Gopkg.toml
	GOBIN=$$(pwd)/build/_output/bin/ go install ./vendor/github.com/aktau/github-release

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
	NODE_NAME=node01 $(GINKGO) $(UNIT_TEST_ARGS) ./pkg/

test/e2e: $(OPERATOR_SDK)
	$(OPERATOR_SDK) test local ./test/e2e \
		--kubeconfig $(KUBECONFIG) \
		--namespace nmstate \
		--no-setup \
		--go-test-flags "$(E2E_TEST_ARGS)"

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

cluster-down: $(CLUSTER_DOWN)
	$(CLUSTER_DOWN)

cluster-clean: $(KUBECTL)
	$(KUBECTL) delete --ignore-not-found -f build/_output/
	$(KUBECTL) delete --ignore-not-found -f deploy/
	$(KUBECTL) delete --ignore-not-found -f deploy/crds/nmstate_v1alpha1_nodenetworkstate_crd.yaml
	$(KUBECTL) delete --ignore-not-found -f deploy/crds/nmstate_v1alpha1_nodenetworkconfigurationpolicy_crd.yaml
	if [[ "$$KUBEVIRT_PROVIDER" =~ ^os-.*$$ ]]; then \
		$(KUBECTL) delete --ignore-not-found -f deploy/openshift/; \
	fi

cluster-sync-resources: $(KUBECTL)
	for resource in $(resources); do \
		$(KUBECTL) apply -f $$resource || exit 1; \
	done
	if [[ "$$KUBEVIRT_PROVIDER" =~ ^os-.*$$ ]]; then \
		$(KUBECTL) apply -f deploy/openshift/; \
	fi

cluster-sync-handler: cluster-sync-resources $(local_handler_manifest)
	IMAGE_REGISTRY=localhost:$(shell $(CLI) ports registry | tr -d '\r') \
				   make push-handler
	# Temporary until image is updated with provisioner that sets this field
	# This field is required by buildah tool
	$(SSH) node01 'sudo sysctl -w user.max_user_namespaces=1024'
	$(KUBECTL) apply -f deploy/crds/nmstate_v1alpha1_nodenetworkstate_crd.yaml
	$(KUBECTL) apply -f deploy/crds/nmstate_v1alpha1_nodenetworkconfigurationpolicy_crd.yaml
	$(KUBECTL) delete --ignore-not-found -f $(local_handler_manifest)
	$(KUBECTL) create -f $(local_handler_manifest)

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
