IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate
MANAGER_IMAGE_NAME ?= kubernetes-nmstate-manager
MANAGER_IMAGE_TAG ?= latest
MANAGER_IMAGE_FULL_NAME ?= $(IMAGE_REGISTRY)/$(IMAGE_REPO)/$(MANAGER_IMAGE_NAME):$(MANAGER_IMAGE_TAG)
GINKGO_EXTRA_ARGS ?=
GINKGO_ARGS ?= -v -r --randomizeAllSpecs --randomizeSuites --race --trace $(GINKGO_EXTRA_ARGS)
GINKGO?= go run ./vendor/github.com/onsi/ginkgo/ginkgo

KUBECONFIG ?= ./cluster/.kubeconfig
LOCAL_REGISTRY ?= registry:5000

all: check manager

check: format vet

format:
	gofmt -d cmd/ pkg/

vet:
	go vet ./cmd/... ./pkg/...

manager:
	operator-sdk build $(MANAGER_IMAGE_FULL_NAME)

gen-k8s:
	operator-sdk generate k8s

push-manager: manager
	docker push $(MANAGER_IMAGE_FULL_NAME)

unit-test:
	$(GINKGO) $(GINKGO_ARGS) ./pkg/

test/e2e:
	operator-sdk test local ./$@ \
		--kubeconfig $(KUBECONFIG) \
		--namespace default \
		--up-local

test/e2e:
	operator-sdk test local ./$@ \
		--kubeconfig $(KUBECONFIG) \
		--namespace default \
		--up-local 

cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-sync:
	./cluster/sync.sh

cluster-clean:
	./cluster/clean.sh

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
