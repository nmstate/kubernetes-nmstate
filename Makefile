all: build

MANIFESTS_SOURCE ?= manifests/templates
MANIFESTS_DESTINATION ?= manifests/examples
NAMESPACE ?= nmstate-default
IMAGE_REGISTRY ?= quay.io/nmstate
IMAGE_TAG ?= latest
PULL_POLICY ?= Always
STATE_HANDLER_IMAGE ?= kubernetes-nmstate-state-handler
POLICY_HANDLER_IMAGE ?= kubernetes-nmstate-configuration-policy-handler
KUBECONFIG ?= $(shell pwd)/cluster/.kubeconfig
FUNC_TEST_ARGS ?= ""

commands = state-handler policy-handler

$(commands):
	go fmt ./cmd/$@
	go vet ./cmd/$@
	go build -o ./bin/$@ ./cmd/$@ 

build: policy-handler state-handler

build:
	cd cmd/state-handler && go fmt && go vet && go build
	cd cmd/policy-handler && go fmt && go vet && go build

docker:
	docker build -f cmd/state-handler/Dockerfile -t $(IMAGE_REGISTRY)/$(STATE_HANDLER_IMAGE):$(IMAGE_TAG) .
	docker build -f cmd/policy-handler/Dockerfile -t $(IMAGE_REGISTRY)/$(POLICY_HANDLER_IMAGE):$(IMAGE_TAG) .

docker-push:
	docker push $(IMAGE_REGISTRY)/$(STATE_HANDLER_IMAGE):$(IMAGE_TAG)
	docker push $(IMAGE_REGISTRY)/$(POLICY_HANDLER_IMAGE):$(IMAGE_TAG)

generate:
	hack/update-codegen.sh

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
	./hack/verify-manifests.sh

functest:
	KUBECONFIG=$(KUBECONFIG) \
	FUNC_TEST_ARGS=$(FUNC_TEST_ARGS) \
		./hack/test-functional.sh

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

.PHONY: policy-handler state-handler build docker docker-push generate manifests check functest dep clean-dep clean-generate clean-manifests cluster-up cluster-sync cluster-clean cluster-down
