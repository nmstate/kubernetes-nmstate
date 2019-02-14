all: build

build:
	cd cmd/state-handler && go fmt && go vet && go build
	cd cmd/policy-handler && go fmt && go vet && go build

IMAGE_REGISTRY ?= yuvalif

docker:
	docker build -f cmd/state-handler/Dockerfile -t $(IMAGE_REGISTRY)/k8s-node-network-state-controller .
	docker build -f cmd/policy-handler/Dockerfile -t $(IMAGE_REGISTRY)/k8s-node-network-configuration-policy-controller .

docker-push: build
	docker push $(IMAGE_REGISTRY)/k8s-node-network-state-controller
	docker push $(IMAGE_REGISTRY)/k8s-node-network-configuration-policy-controller

generate:
	hack/update-codegen.sh

MANIFESTS_SOURCE ?= manifests/templates
MANIFESTS_DESTINATION ?= manifests/examples
NAMESPACE ?= nmstate-default
IMAGE_REGISTRY ?= yuvalif
IMAGE_TAG ?= latest
PULL_POLICY ?= Always
STATE_HANDLER_IMAGE ?= k8s-node-network-state-controller

manifests:
	MANIFESTS_SOURCE=$(MANIFESTS_SOURCE) \
	MANIFESTS_DESTINATION=$(MANIFESTS_DESTINATION) \
	NAMESPACE=$(NAMESPACE) \
	IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
	IMAGE_TAG=$(IMAGE_TAG) \
	PULL_POLICY=$(PULL_POLICY) \
	STATE_HANDLER_IMAGE=$(STATE_HANDLER_IMAGE) \
		hack/generate-manifests.sh

check:
	./hack/verify-codegen.sh
	./hack/verify-fmt.sh
	./hack/verify-vet.sh

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

.PHONY: build docker docker-push generate manifests check dep clean-dep clean-generate clean-manifests cluster-up cluster-sync cluster-clean cluster-down
