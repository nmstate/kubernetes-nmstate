all: build

build:
	cd cmd/client && go fmt && go vet && go build
	cd cmd/state-controller && go fmt && go vet && go build
	cd cmd/policy-controller && go fmt && go vet && go build

IMAGE_REGISTRY ?= yuvalif

docker: build
	cd cmd/client && docker build -t $(IMAGE_REGISTRY)/k8s-node-net-conf-client .
	cd cmd/state-controller && docker build -t $(IMAGE_REGISTRY)/k8s-node-network-state-controller .

docker-push: build
	docker push $(IMAGE_REGISTRY)/k8s-node-net-conf-client
	docker push $(IMAGE_REGISTRY)/k8s-node-network-state-controller

generate:
	hack/update-codegen.sh

MANIFESTS_SOURCE ?= manifests/templates
MANIFESTS_DESTINATION ?= manifests/examples
NAMESPACE ?= nmstate-default
IMAGE_REGISTRY ?= yuvalif
IMAGE_TAG ?= latest
PULL_POLICY ?= Always
STATE_CLIENT_IMAGE ?= k8s-node-net-conf-client
STATE_CONTROLLER_IMAGE ?= k8s-node-network-state-controller

manifests:
	MANIFESTS_SOURCE=$(MANIFESTS_SOURCE) \
	MANIFESTS_DESTINATION=$(MANIFESTS_DESTINATION) \
	NAMESPACE=$(NAMESPACE) \
	IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
	IMAGE_TAG=$(IMAGE_TAG) \
	PULL_POLICY=$(PULL_POLICY) \
	STATE_CLIENT_IMAGE=$(STATE_CLIENT_IMAGE) \
	STATE_CONTROLLER_IMAGE=$(STATE_CONTROLLER_IMAGE) \
		hack/generate-manifests.sh

check:
	./hack/verify-fmt.sh

test:
	@echo "==========Running Policy Client Test..."
	hack/test-client-policy.sh
	@echo "==========Running State Client Test..."
	hack/test-client-state.sh
	@echo "==========Running State Controller Test..."
	hack/test-controller-state.sh

dep:
	dep ensure -v

clean-dep:
	rm -f ./Gopkg.lock
	rm -rf ./vendor

clean-generate:
	rm -f pkg/apis/nmstate.io/v1/zz_generated.deepcopy.go
	rm -rf pkg/client

clean-manifests:
	rm -rf $(MANIFESTS_DESTINATION)

cluster-up:
	./cluster/up.sh

cluster-sync:
	./cluster/sync.sh

cluster-clean:
	./cluster/clean.sh

cluster-down:
	./cluster/down.sh

.PHONY: build docker docker-push generate manifests check test dep clean-dep clean-generate clean-manifests cluster-up cluster-sync cluster-clean cluster-down
