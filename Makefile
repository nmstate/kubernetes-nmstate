IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate

MANAGER_IMAGE_NAME ?= kubernetes-nmstate-manager
MANAGER_IMAGE_TAG ?= latest
MANAGER_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(MANAGER_IMAGE_NAME):$(MANAGER_IMAGE_TAG)
MANAGER_IMAGE ?= $(IMAGE_REGISTRY)/$(MANAGER_IMAGE_FULL_NAME)

HANDLER_IMAGE_NAME ?= kubernetes-nmstate-handler
HANDLER_IMAGE_TAG ?= latest
HANDLER_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(HANDLER_IMAGE_NAME):$(HANDLER_IMAGE_TAG)
HANDLER_IMAGE ?= $(IMAGE_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)

GINKGO_EXTRA_ARGS ?=
GINKGO_ARGS ?= -v -r --randomizeAllSpecs --randomizeSuites --race --trace $(GINKGO_EXTRA_ARGS)
GINKGO?= go run ./vendor/github.com/onsi/ginkgo/ginkgo
OPERATOR_SDK ?= go run ./vendor/github.com/operator-framework/operator-sdk/cmd/operator-sdk
LOCAL_REGISTRY ?= registry:5000

export KUBEVIRT_PROVIDER ?= k8s-1.11.0
export KUBEVIRT_NUM_NODES ?= 1

CLUSTER_DIR ?= kubevirtci/cluster-up/
KUBECONFIG ?= kubevirtci/_ci-configs/$(KUBEVIRT_PROVIDER)/.kubeconfig
KUBECTL ?= $(CLUSTER_DIR)/kubectl.sh
CLUSTER_UP ?= $(CLUSTER_DIR)/up.sh
CLUSTER_DOWN ?= $(CLUSTER_DIR)/down.sh
CLI ?= $(CLUSTER_DIR)/cli.sh
SSH ?= $(CLI) ssh

local_handler_manifest = build/_output/handler.local.yaml
local_manager_manifest = build/_output/manager.local.yaml

resources = deploy/service_account.yaml deploy/role.yaml deploy/role_binding.yaml

all: check manager

check: format vet

format:
	gofmt -d cmd/ pkg/

vet:
	go vet ./cmd/... ./pkg/...

manager:
	$(OPERATOR_SDK) build $(MANAGER_IMAGE)

manager-up:
	$(KUBECTL) apply -f deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml
	OPERATOR_NAME=nmstate-manager \
				  $(OPERATOR_SDK) up local --kubeconfig $(KUBECONFIG)

handler:
	docker build -t $(HANDLER_IMAGE) build/handler

gen-k8s:
	$(OPERATOR_SDK) generate k8s

push-manager: manager
	docker push $(MANAGER_IMAGE)

push-handler: handler
	docker push $(HANDLER_IMAGE)

unit-test:
	$(GINKGO) $(GINKGO_ARGS) ./pkg/

test/local/e2e:
	OPERATOR_NAME=nmstate-manager\
		$(OPERATOR_SDK) test local ./test/e2e \
			--kubeconfig $(KUBECONFIG) \
			--namespace default \
			--global-manifest deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml \
			--up-local \
			--go-test-flags "-test.v -ginkgo.v"

test/cluster/e2e:
	$(OPERATOR_SDK) test local ./test/e2e \
		--kubeconfig $(KUBECONFIG) \
		--namespace default \
		--no-setup \
		--go-test-flags "-test.v -ginkgo.v"


$(local_handler_manifest): deploy/handler.yaml
	mkdir -p $(dir $@)
	sed "s#REPLACE_IMAGE#$(LOCAL_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)#" \
		deploy/handler.yaml > $@

$(local_manager_manifest): deploy/operator.yaml
	mkdir -p $(dir $@)
	sed "s#REPLACE_IMAGE#$(LOCAL_REGISTRY)/$(MANAGER_IMAGE_FULL_NAME)#" \
		deploy/operator.yaml > $@

$(CLUSTER_DIR)/%: kubevirtci.version
	rm -rf kubevirtci
	git clone https://github.com/kubevirt/kubevirtci
	cd kubevirtci && git checkout $$(cat  ../kubevirtci.version)

cluster-up: $(CLUSTER_UP)
	$(CLUSTER_UP)
	$(SSH) node01 -- sudo yum install -y NetworkManager NetworkManager-ovs
	$(SSH) node01 -- sudo systemctl daemon-reload
	$(SSH) node01 -- sudo systemctl restart NetworkManager

cluster-down: $(CLUSTER_DOWN)
	$(CLUSTER_DOWN)

cluster-clean: $(KUBECTL)
	$(KUBECTL) delete --ignore-not-found -f build/_output/
	$(KUBECTL) delete --ignore-not-found -f deploy/
	$(KUBECTL) delete --ignore-not-found -f deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml
	if [[ "$$KUBEVIRT_PROVIDER" =~ ^os-.*$$ ]]; then \
		$(KUBECTL) delete --ignore-not-found -f deploy/openshift/; \
	fi

cluster-sync-resources: $(KUBECTL)
	for resource in $(resources); do \
		$(KUBECTL) apply -f $$resource; \
	done
	if [[ "$$KUBEVIRT_PROVIDER" =~ ^os-.*$$ ]]; then \
		$(KUBECTL) apply -f deploy/openshift/; \
	fi


cluster-sync-handler: cluster-sync-resources $(local_handler_manifest) $(CLI) $(KUBECTL)
	IMAGE_REGISTRY=localhost:$(shell $(CLI) ports registry | tr -d '\r') \
		make push-handler
	$(SSH) node01 'sudo docker pull $(LOCAL_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)'
	# Temporary until image is updated with provisioner that sets this field
	# This field is required by buildah tool
	$(SSH) node01 'sudo sysctl -w user.max_user_namespaces=1024'
	$(KUBECTL) delete --ignore-not-found -f $(local_handler_manifest)
	$(KUBECTL) create -f $(local_handler_manifest)

cluster-sync-manager: cluster-sync-resources $(CLI) $(local_manager_manifest)
	IMAGE_REGISTRY=localhost:$(shell $(CLI) ports registry | tr -d '\r') \
		make push-manager
	$(SSH) node01 'sudo docker pull $(LOCAL_REGISTRY)/$(MANAGER_IMAGE_FULL_NAME)'
	# Temporary until image is updated with provisioner that sets this field
	# This field is required by buildah tool
	$(SSH) node01 'sudo sysctl -w user.max_user_namespaces=1024'
	$(KUBECTL) apply -f deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml
	$(KUBECTL) delete --ignore-not-found -f $(local_manager_manifest)
	$(KUBECTL) create -f $(local_manager_manifest)

cluster-sync: cluster-sync-handler cluster-sync-manager

.PHONY: \
	all \
	check \
	format \
	vet \
	manager \
	push-manager \
	test-unit \
	test/local/e2e \
	test/cluster/e2e \
	cluster-up \
	cluster-down \
	cluster-sync-resources \
	cluster-sync-manager \
	cluster-sync-handler \
	cluster-sync \
	cluster-clean
