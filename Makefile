MANAGER_IMAGE ?= kubernetes-nmstate-manager

all: check manager

check: format vet 

format:
	gofmt -d cmd/ pkg/

vet:
	go vet ./cmd/... ./pkg/...

manager:
	operator-sdk build $(MANAGER_IMAGE)

.PHONY: \
	all \
	check \
	format \
	vet \
	manager
