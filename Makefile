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

KUBECONFIG ?= ./cluster/.kubeconfig
LOCAL_REGISTRY ?= registry:5000

local_handler_manifest = build/_output/handler.local.yaml

all: check manager

check: format vet

format:
	gofmt -d cmd/ pkg/

vet:
	go vet ./cmd/... ./pkg/...

manager:
	operator-sdk build $(MANAGER_IMAGE)

manager-up:
	operator-sdk up local --kubeconfig $(KUBECONFIG)

handler:
	docker build -t $(HANDLER_IMAGE) build/handler

gen-k8s:
	operator-sdk generate k8s

push-manager: manager
	docker push $(MANAGER_IMAGE)

push-handler: handler
	docker push $(HANDLER_IMAGE)

unit-test:
	$(GINKGO) $(GINKGO_ARGS) ./pkg/

test/e2e:
	operator-sdk test local ./$@ \
		--kubeconfig $(KUBECONFIG) \
		--namespace default \
		--global-manifest deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml \
		--up-local

$(local_handler_manifest): deploy/handler.yaml
	sed "s#REPLACE_IMAGE#$(LOCAL_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)#" \
		deploy/handler.yaml > $@

cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-clean:
	./cluster/clean.sh

cluster-sync: $(local_handler_manifest)
	IMAGE_REGISTRY=localhost:$(shell ./cluster/cli.sh ports registry | tr -d '\r') \
		make push-handler
	./cluster/cli.sh ssh node01 'sudo docker pull $(LOCAL_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)'
	# Temporary until image is updated with provisioner that sets this field
	# This field is required by buildah tool
	./cluster/cli.sh ssh node01 'sudo sysctl -w user.max_user_namespaces=1024'
	./cluster/kubectl.sh apply -f $(local_handler_manifest)


.PHONY: \
	all \
	check \
	format \
	vet \
	manager \
	push-manager \
	test-unit \
	test/e2e \
	cluster-up \
	cluster-down \
	cluster-sync \
	cluster-clean
