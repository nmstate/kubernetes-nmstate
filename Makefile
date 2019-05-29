IMAGE_REPO ?= quay.io/nmstate
IMAGE_TAG ?= latest
MANAGER_IMAGE ?= kubernetes-nmstate-manager

all:
	@echo Hello

check: check-gofmt check-vet

check-gofmt:
	./hack/check-gofmt.sh

check-vet:
	./hack/check-vet.sh

build-manager:
	operator-sdk build $(IMAGE_REPO)/$(MANAGER_IMAGE):$(IMAGE_TAG)

push-manager:
	docker push $(IMAGE_REPO)/$(MANAGER_IMAGE):$(IMAGE_TAG)

cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-sync:
	./cluster/sync.sh

cluster-clean:
	./cluster/clean.sh

.PHONY:
	check \
	check-gofmt \
	check-vet \
	build-manager \
	cluster-up \
	cluster-down \
	cluster-sync \
	cluster-clean
