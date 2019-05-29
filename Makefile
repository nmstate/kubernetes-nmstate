MANAGER_IMAGE ?= kubernetes-nmstate-manager

all:
	@echo Hello

check: check-gofmt check-vet

check-gofmt:
	./hack/check-gofmt.sh

check-vet:
	./hack/check-vet.sh

build-manager:
	operator-sdk build $(MANAGER_IMAGE)

.PHONY:
	check \
	check-gofmt \
	check-vet \
	build-manager
