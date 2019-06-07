IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate

MANAGER_IMAGE_NAME ?= kubernetes-nmstate-manager
MANAGER_IMAGE_TAG ?= latest
MANAGER_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(MANAGER_IMAGE_NAME):$(MANAGER_IMAGE_TAG)
MANAGER_IMAGE ?= $(IMAGE_REGISTRY)/$(MANAGER_IMAGE_FULL_NAME)

GINKGO_EXTRA_ARGS ?=
GINKGO_ARGS ?= -v -r --randomizeAllSpecs --randomizeSuites --race --trace $(GINKGO_EXTRA_ARGS)
GINKGO?= go run ./vendor/github.com/onsi/ginkgo/ginkgo

KUBECONFIG ?= ./cluster/.kubeconfig
# TODO: use operator-sdk from vendor/
OPERATOR_SDK ?= operator-sdk
LOCAL_REGISTRY ?= registry:5000
KUBECTL ?= ./cluster/kubectl.sh

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

gen-k8s:
	$(OPERATOR_SDK) generate k8s

push-manager: manager
	docker push $(MANAGER_IMAGE)

unit-test:
	$(GINKGO) $(GINKGO_ARGS) ./pkg/

test/local/e2e:
	OPERATOR_NAME=nmstate-manager\
		$(OPERATOR_SDK) test local ./test/e2e \
			--kubeconfig $(KUBECONFIG) \
			--namespace default \
			--global-manifest deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml \
			--up-local

test/cluster/e2e:
	$(OPERATOR_SDK) test local ./test/e2e \
		--kubeconfig $(KUBECONFIG) \
		--namespace default \
		--no-setup


$(local_manager_manifest): deploy/operator.yaml
	sed "s#REPLACE_IMAGE#$(LOCAL_REGISTRY)/$(MANAGER_IMAGE_FULL_NAME)#" \
		deploy/operator.yaml > $@


cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-clean:
	./cluster/clean.sh

cluster-sync-resources:
	for resource in $(resources); do \
		$(KUBECTL) apply -f $$resource; \
	done


cluster-sync-manager: cluster-sync-resources $(local_manager_manifest)
	IMAGE_REGISTRY=localhost:$(shell ./cluster/cli.sh ports registry | tr -d '\r') \
		make push-manager
	./cluster/cli.sh ssh node01 'sudo docker pull $(LOCAL_REGISTRY)/$(MANAGER_IMAGE_FULL_NAME)'
	# Temporary until image is updated with provisioner that sets this field
	# This field is required by buildah tool
	./cluster/cli.sh ssh node01 'sudo sysctl -w user.max_user_namespaces=1024'
	$(KUBECTL) apply -f deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml
	$(KUBECTL) delete --ignore-not-found -f $(local_manager_manifest)
	$(KUBECTL) create -f $(local_manager_manifest)

cluster-sync: cluster-sync-manager

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
	cluster-sync \
	cluster-clean
