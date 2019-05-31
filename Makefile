IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate
MANAGER_IMAGE_NAME ?= kubernetes-nmstate-manager
MANAGER_IMAGE_TAG ?= latest
MANAGER_IMAGE_FULL_NAME ?= $(IMAGE_REGISTRY)/$(IMAGE_REPO)/$(MANAGER_IMAGE_NAME):$(MANAGER_IMAGE_TAG)

all: check manager

check: format vet

format:
	gofmt -d cmd/ pkg/

vet:
	go vet ./cmd/... ./pkg/...

manager:
	operator-sdk build $(MANAGER_IMAGE_FULL_NAME)

push-manager:
	docker push $(MANAGER_IMAGE_FULL_NAME)

gen-k8s:
	operator-sdk generate k8s

unit-test:
	ginkgo build ./pkg/apis/nmstate/v1
	ginkgo ./pkg/apis/nmstate/v1

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
	test/unit \
	cluster-up \
	cluster-down \
	cluster-sync \
	cluster-clean
