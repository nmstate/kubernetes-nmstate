CONTROLLER_GEN = go run sigs.k8s.io/controller-tools/cmd/controller-gen

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

export GOFLAGS=-mod=mod

gen-k8s:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

gen-crds:
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=../deploy/crds
