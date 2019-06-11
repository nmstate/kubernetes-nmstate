IMAGE_REGISTRY ?= quay.io
IMAGE_REPO ?= nmstate

HANDLER_IMAGE_NAME ?= kubernetes-nmstate-handler
HANDLER_IMAGE_TAG ?= latest
HANDLER_IMAGE_FULL_NAME ?= $(IMAGE_REPO)/$(HANDLER_IMAGE_NAME):$(HANDLER_IMAGE_TAG)
HANDLER_IMAGE ?= $(IMAGE_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)

GINKGO_EXTRA_ARGS ?=
GINKGO_ARGS ?= -v -r --randomizeAllSpecs --randomizeSuites --race --trace $(GINKGO_EXTRA_ARGS)
GINKGO?= go run ./vendor/github.com/onsi/ginkgo/ginkgo

E2E_TEST_EXTRA_ARGS ?=
E2E_TEST_ARGS ?= $(strip -test.v -ginkgo.v $(E2E_TEST_EXTRA_ARGS))

KUBECONFIG ?= ./cluster/.kubeconfig
# TODO: use operator-sdk from vendor/
OPERATOR_SDK ?= go run ./vendor/github.com/operator-framework/operator-sdk/cmd/operator-sdk
LOCAL_REGISTRY ?= registry:5000
KUBECTL ?= ./cluster/kubectl.sh

local_handler_manifest = build/_output/handler.local.yaml

resources = deploy/service_account.yaml deploy/role.yaml deploy/role_binding.yaml

all: check handler

check: format vet

format:
	gofmt -d cmd/ pkg/

vet:
	go vet ./cmd/... ./pkg/...

handler:
	$(OPERATOR_SDK) build $(HANDLER_IMAGE)

handler-up:
	$(KUBECTL) apply -f deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml
	OPERATOR_NAME=nmstate-handler \
				  $(OPERATOR_SDK) up local --kubeconfig $(KUBECONFIG)

gen-k8s:
	$(OPERATOR_SDK) generate k8s

push-handler: handler
	docker push $(HANDLER_IMAGE)

unit-test:
	$(GINKGO) $(GINKGO_ARGS) ./pkg/

test/local/e2e:
	OPERATOR_NAME=nmstate-handler\
		$(OPERATOR_SDK) test local ./test/e2e \
			--kubeconfig $(KUBECONFIG) \
			--namespace default \
			--global-manifest deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml \
			--up-local \
			--go-test-flags "$(E2E_TEST_ARGS)"

test/cluster/e2e:
	$(OPERATOR_SDK) test local ./test/e2e \
		--kubeconfig $(KUBECONFIG) \
		--namespace default \
		--no-setup \
		--go-test-flags "$(E2E_TEST_ARGS)"


$(local_handler_manifest): deploy/operator.yaml
	mkdir -p $$(dirname $@)
	sed "s#REPLACE_IMAGE#$(LOCAL_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)#" \
		deploy/operator.yaml > $@


cluster-up:
	./cluster/up.sh

cluster-down:
	./cluster/down.sh

cluster-clean:
	$(KUBECTL) delete --ignore-not-found -f build/_output/
	$(KUBECTL) delete --ignore-not-found -f deploy/
	$(KUBECTL) delete --ignore-not-found -f deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml
	if [[ "$$KUBEVIRT_PROVIDER" =~ ^os-.*$$ ]]; then \
		$(KUBECTL) delete --ignore-not-found -f deploy/openshift/; \
	fi

cluster-sync-resources:
	for resource in $(resources); do \
		$(KUBECTL) apply -f $$resource; \
	done
	if [[ "$$KUBEVIRT_PROVIDER" =~ ^os-.*$$ ]]; then \
		$(KUBECTL) apply -f deploy/openshift/; \
	fi

cluster-sync-handler: cluster-sync-resources $(local_handler_manifest)
	IMAGE_REGISTRY=localhost:$(shell ./cluster/cli.sh ports registry | tr -d '\r') \
		make push-handler
	./cluster/cli.sh ssh node01 'sudo docker pull $(LOCAL_REGISTRY)/$(HANDLER_IMAGE_FULL_NAME)'
	# Temporary until image is updated with provisioner that sets this field
	# This field is required by buildah tool
	./cluster/cli.sh ssh node01 'sudo sysctl -w user.max_user_namespaces=1024'
	$(KUBECTL) apply -f deploy/crds/nmstate_v1_nodenetworkstate_crd.yaml
	$(KUBECTL) delete --ignore-not-found -f $(local_handler_manifest)
	$(KUBECTL) create -f $(local_handler_manifest)

cluster-sync: cluster-sync-handler

.PHONY: \
	all \
	check \
	format \
	vet \
	handler \
	push-handler \
	test-unit \
	test/local/e2e \
	test/cluster/e2e \
	cluster-up \
	cluster-down \
	cluster-sync-resources \
	cluster-sync-handler \
	cluster-sync \
	cluster-clean
