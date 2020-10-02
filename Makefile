# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

BIN_DIR=$(CURDIR)/_bin/

export GOFLAGS=-mod=vendor
export GOPROXY=direct
export GOSUMDB=off
export GOROOT=$(BIN_DIR)/go/
export GOBIN=$(BIN_DIR)
export PATH := $(BIN_DIR):$(GOROOT)/bin:$(PATH)

OPENAPI_GEN ?= $(GOBIN)/openapi-gen
CONTROLLER_GEN ?= $(GOBIN)/controller-gen
KUSTOMIZE ?= $(GOBIN)/kustomize
GINKGO?= $(GOBIN)/ginkgo
export GITHUB_RELEASE ?= $(GOBIN)/github-release
export RELEASE_NOTES ?= $(GOBIN)/release-notes
GOFMT := $(GOBIN)/gofmt
export GO := $(GOROOT)/bin/go

all: manager

$(GO):
	hack/install-go.sh $(BIN_DIR)

$(GINKGO): go.mod
	$(MAKE) tools
$(OPENAPI_GEN): go.mod
	$(MAKE) tools
$(GITHUB_RELEASE): go.mod
	$(MAKE) tools
$(RELEASE_NOTES): go.mod
	$(MAKE) tools
$(CONTROLLER_GEN): go.mod
	$(MAKE) tools
$(KUSTOMIZE): go.mod
	$(MAKE) tools

E2E_TEST_TIMEOUT ?= 40m
unit_test_args ?=  -r -keepGoing --randomizeAllSpecs --randomizeSuites --race --trace $(UNIT_TEST_ARGS)
e2e_test_args = -v -timeout=$(E2E_TEST_TIMEOUT) -slowSpecThreshold=60 $(E2E_TEST_ARGS)

# Run tests
test-unit: $(GO)
	INTERFACES_FILTER="" NODE_NAME=node01 $(GINKGO) $(unit_test_args) ./controllers ./api ./pkg ... -coverprofile cover.out

# Run handler functests
test-e2e-handler: $(GINKGO)
	$(GINKGO) $(e2e_test_args) ./test/e2e/handler ...

# Run operator functests
test-e2e-operator: $(GINKGO)
	$(GINKGO) $(e2e_test_args) ./test/e2e/operator ...

# Build manager binary
manager: $(GO) generate fmt vet
	$(GO) build -o $(BIN_DIR)/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: $(GO) generate fmt vet manifests
	$(GO) run ./main.go

# Install CRDs into a cluster
install: manifests $(KUSTOMIZE)
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests $(KUSTOMIZE)
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found -f -

resources: $(KUSTOMIZE)
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests $(KUSTOMIZE)
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role paths="./..." output:crd:artifacts:config=config/crd/bases

# Run $(GO) fmt against code
fmt: $(GO)
	$(GO) fmt ./test/... ./pkg/... ./controllers/... ./api/...
	$(GO) fmt main.go

# Run $(GO) vet against code
vet: $(GO)
	$(GO) vet ./test/... ./pkg/... ./controllers/... ./api/...
	$(GO) vet main.go

# Generate code
generate: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: manager test
	docker build . -f Dockerfile.operator -t ${IMG}

# Push the docker image
docker-push: IMG=localhost:$(shell ./cluster/cli.sh ports registry | tr -d '\r')/controller:latest
docker-push: docker-build
	docker push ${IMG}

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests $(KUSTOMIZE)
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: cluster-up
cluster-up:
	./cluster/up.sh

.PHONY: cluster-down
cluster-down:
	./cluster/down.sh

.PHONY: cluster-sync
cluster-sync: docker-push
cluster-sync: IMG=registry:5000/controller:latest
cluster-sync: install deploy

.PHONY: cluster-clean
cluster-clean: uninstall
	kubectl delete --ignore-not-found namespace nmstate

.PHONY: vendor
vendor: $(GO)
	$(GO) mod tidy
	$(GO) mod vendor

.PHONY: tools
tools: $(GO)
	./hack/install-tools.sh
