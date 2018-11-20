all: build

build:
	cd cmd/client && go fmt && go vet && go build
	cd cmd/state-controller && go fmt && go vet && go build
	cd cmd/policy-controller && go fmt && go vet && go build

GENERATED_MANIFEST_DIR=manifests/generated

generate:
	hack/update-codegen.sh
	cd tools && go fmt && go build -o crd-generator
	mkdir -p manifests/generated
	tools/crd-generator -crd-type net-state > $(GENERATED_MANIFEST_DIR)/net-state-crd.yaml
	tools/crd-generator -crd-type net-conf > $(GENERATED_MANIFEST_DIR)/net-conf-crd.yaml
	tools/crd-generator -crd-type net-conf-sample > $(GENERATED_MANIFEST_DIR)/net-conf-sample.yaml
	tools/crd-generator -crd-type net-state-sample > $(GENERATED_MANIFEST_DIR)/net-state-sample.yaml
	tools/crd-generator -crd-type net-state-ethernet > $(GENERATED_MANIFEST_DIR)/net-state-ethernet.yaml

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
	rm -rf $(GENERATED_MANIFEST_DIR)
