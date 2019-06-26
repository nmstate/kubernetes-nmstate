IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate

TAG ?= $(shell grep = version/version.go | sed -r 's/.*= \"(.*)"$/\1/g')
HANDLER_IMAGE_NAME ?= kubernetes-nmstate-handler
#HANDLER_IMAGE_SUFFIX ?= :latest
HANDLER_IMAGE_SUFFIX ?=
HANDLER_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(HANDLER_IMAGE_NAME)$(HANDLER_IMAGE_SUFFIX)
HANDLER_IMAGE ?= $(IMAGE_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)

GINKGO_EXTRA_ARGS ?=
GINKGO_ARGS ?= -v -r --randomizeAllSpecs --randomizeSuites --race --trace $(GINKGO_EXTRA_ARGS)
GINKGO?= build/_output/bin/ginkgo

E2E_TEST_EXTRA_ARGS ?=
E2E_TEST_ARGS ?= $(strip -test.v -ginkgo.v $(E2E_TEST_EXTRA_ARGS))

OPERATOR_SDK ?= build/_output/bin/operator-sdk
GITHUB_RELEASE ?= build/_output/bin/github-release
LOCAL_REGISTRY ?= registry:5000

export KUBEVIRT_PROVIDER ?= k8s-1.13.3
export KUBEVIRT_NUM_NODES ?= 1
export KUBEVIRT_NUM_SECONDARY_NICS ?= 1

CLUSTER_DIR ?= kubevirtci/cluster-up/
KUBECONFIG ?= kubevirtci/_ci-configs/$(KUBEVIRT_PROVIDER)/.kubeconfig
KUBECTL ?= $(CLUSTER_DIR)/kubectl.sh
CLUSTER_UP ?= $(CLUSTER_DIR)/up.sh
CLUSTER_DOWN ?= $(CLUSTER_DIR)/down.sh
CLI ?= $(CLUSTER_DIR)/cli.sh
SSH ?= $(CLUSTER_DIR)/ssh.sh

install_kubevirtci := hack/install-kubevirtci.sh
local_handler_manifest = build/_output/handler.local.yaml
operator_manifest = build/_output/operator.yaml
version = build/_output/version
description = build/_output/description

resources = deploy/service_account.yaml deploy/role.yaml deploy/role_binding.yaml

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

push-handler: handler
	docker push $(HANDLER_IMAGE)

test/unit: $(GINKGO)
	$(GINKGO) $(GINKGO_ARGS) ./pkg/

test/e2e: $(OPERATOR_SDK)
	$(OPERATOR_SDK) test local ./test/e2e \
		--kubeconfig $(KUBECONFIG) \
		--namespace default \
		--no-setup \
		--go-test-flags "$(E2E_TEST_ARGS)"

$(local_handler_manifest): deploy/operator.yaml
	mkdir -p $(dir $@)
	sed "s#REPLACE_IMAGE#$(LOCAL_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)#" \
		deploy/operator.yaml > $@

$(operator_manifest): deploy/operator.yaml
	mkdir -p $(dir $@)
	sed "s#REPLACE_IMAGE#$(HANDLER_IMAGE)#" \
		deploy/operator.yaml > $@

$(): deploy/operator.yaml
	mkdir -p $(dir $@)
	sed "s#REPLACE_IMAGE#$(LOCAL_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)#" \
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

cluster-sync-handler: IMAGE_REGISTRY = localhost:$(shell $(CLI) ports registry | tr -d '\r')
cluster-sync-handler: cluster-sync-resources $(local_handler_manifest) push-handler
	# Temporary until image is updated with provisioner that sets this field
	# This field is required by buildah tool
	$(SSH) node01 'sudo sysctl -w user.max_user_namespaces=1024'
	$(KUBECTL) apply -f deploy/crds/nmstate_v1alpha1_nodenetworkstate_crd.yaml
	$(KUBECTL) delete --ignore-not-found -f $(local_handler_manifest)
	$(KUBECTL) create -f $(local_handler_manifest)

$(version): version/version.go
	grep = version/version.go | sed -r 's/.*= \"(.*)"$$/v\1/g' \
		   > $(version)

$(description): version/description
	mkdir -p $(dir $@)
	sed "s#HANDLER_IMAGE#$(HANDLER_IMAGE)#" \
		version/description > $@

release: $(version)
release: HANDLER_IMAGE_SUFFIX = :$(file < $(version))
release: $(operator_manifest) push-handler $(description)
	DESCRIPTION=$(description) \
	HANDLER_IMAGE=$(HANDLER_IMAGE) \
	GITHUB_RELEASE=$(GITHUB_RELEASE) \
	TAG=$(file < $(version)) \
				   hack/release.sh \
				   		$(resources) \
						$(operator_manifest) \
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
