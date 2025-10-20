# AGENTS.md

This file provides guidance to AI coding assistants (such as Claude Code, GitHub Copilot, Google Gemini, etc.) when working with code in this repository.

## Project Overview

kubernetes-nmstate is a Kubernetes operator that provides declarative node network configuration driven through Kubernetes API. It uses [nmstate](https://nmstate.io/) to perform state-driven network configuration on cluster nodes and report their current state. The only external dependency is NetworkManager running on nodes.

## Architecture

### Dual Component Design

The system consists of two main components:

1. **Handler** (cmd/handler/): Runs as a DaemonSet on each node
   - Performs actual network configuration on nodes via NetworkManager
   - Watches for NodeNetworkConfigurationPolicy resources targeting its node
   - Reports node network state through NodeNetworkState resources
   - Uses event filtering to only process events for its own node
   - Implements reconciliation controllers for network state

2. **Operator** (cmd/operator/): Cluster-scoped operator
   - Manages the overall deployment of handlers
   - Reconciles the NMState CR to deploy/update handler DaemonSet
   - Manages operator lifecycle

This differs from typical operators which run as Deployments - the handler is deployed as a DaemonSet with per-node event filtering.

### Custom Resource Definitions (CRDs)

- **NMState** (nmstate.io/v1): Operator deployment configuration
- **NodeNetworkConfigurationPolicy** (NNCP): Cluster-scoped desired network configuration
- **NodeNetworkConfigurationEnactment** (NNCE): Per-node policy application tracking
- **NodeNetworkState** (NNS): Reports current network state of nodes

Located in: `api/v1` and `api/v1beta1`

### Key Packages

- `pkg/state/`: Network state management and nmstate integration
- `pkg/enactmentstatus/`: Tracks policy enactment status per node
- `pkg/policyconditions/`: Policy condition management
- `pkg/nmstatectl/`: Interface to nmstatectl CLI
- `pkg/node/`: Node-specific operations
- `pkg/webhook/`: Admission webhooks for CR validation
- `pkg/bridge/`: Bridge network configuration
- `pkg/selectors/`: Node selector matching logic
- `controllers/handler/`: Handler reconciliation controllers
- `controllers/operator/`: Operator reconciliation controllers

## Build Commands

```bash
# Build handler container image
make handler

# Build operator container image
make operator

# Build both images
make all

# Push handler image to registry
make push-handler

# Push operator image to registry
make push-operator

# Push both images
make push
```

Container image configuration via environment variables:
- `IMAGE_REGISTRY`: Container registry (default: quay.io)
- `IMAGE_REPO`: Repository name (default: nmstate)
- `HANDLER_IMAGE_NAME`: Handler image name (default: kubernetes-nmstate-handler)
- `HANDLER_IMAGE_TAG`: Handler image tag (default: latest)
- `OPERATOR_IMAGE_NAME`: Operator image name (default: kubernetes-nmstate-operator)
- `OPERATOR_IMAGE_TAG`: Operator image tag (default: latest)
- `IMAGE_BUILDER`: podman or docker (auto-detected)

## Code Generation

```bash
# Generate CRDs from api/ types
make gen-crds

# Generate Kubernetes client code
make gen-k8s

# Generate RBAC manifests
make gen-rbac

# Run all generators
make generate

# Verify generated code is up to date
make check-gen
```

**Important**: Always run `make generate` after modifying types in `api/v1` or `api/v1beta1`, or after changing controller RBAC markers.

## Testing

### Unit Tests

```bash
# Run all unit tests
make test/unit

# Run unit tests for specific package
make test/unit WHAT=./pkg/state/...

# Run API unit tests
make test/unit/api
```

Unit tests use Ginkgo v2 with parameters: `-r -keep-going --randomize-all --randomize-suites --race --trace`

### E2E Tests

```bash
# Run all e2e tests (requires running cluster with kubernetes-nmstate)
make test-e2e

# Run only handler e2e tests
make test-e2e-handler

# Run only operator e2e tests
make test-e2e-operator

# Run specific tests by regex
make test-e2e E2E_TEST_ARGS='-ginkgo.focus=NodeSelector'

# Exclude specific tests
make test-e2e E2E_TEST_ARGS='--ginkgo.skip="Simple\ OVS*"'
```

E2E tests are located in `test/e2e/` and require a Kubernetes cluster with the operator deployed.

### Static Checks

```bash
# Run all checks (lint, vet, whitespace, gofmt, promlint)
make check

# Run individual checks
make lint
make vet
make whitespace-check
make gofmt-check
make promlint-check

# Auto-format code
make format          # runs whitespace-format and gofmt
make gofmt           # format go files
make whitespace-format
```

## Local Development Cluster

```bash
# Start local virtualized cluster (requires kubevirtci)
make cluster-up

# Deploy/update operator and handler to cluster
make cluster-sync

# Deploy only operator changes
make cluster-sync-operator

# Stop cluster
make cluster-down

# Clean cluster state
make cluster-clean
```

Cluster configuration via environment variables:
- `KUBEVIRT_PROVIDER`: Kubernetes version (default: k8s-1.32)
- `KUBEVIRT_NUM_NODES`: Number of nodes (default: 3)
- `KUBEVIRT_NUM_SECONDARY_NICS`: Secondary NICs per node (default: 2)
- `KUBECONFIG`: Path to kubeconfig (auto-detected via ./cluster/kubeconfig.sh)

Network interface names vary by provider:
- k8s providers: eth0, eth1, eth2
- Other providers: ens3, ens8, ens9

## OpenShift Development

For OpenShift-specific development, see `developing-on-ocp.md`. Key commands:

```bash
# Build and deploy operator via OLM bundle
IMAGE_REPO=<quay-username> KUBECONFIG=<path> make ocp-build-and-deploy-bundle

# Update OCP bundle manifests
make ocp-update-bundle-manifests

# Verify OCP bundle is up to date
make check-ocp-bundle

# Uninstall OCP bundle
make ocp-uninstall-bundle

# Run OCP e2e tests
make test-e2e-handler-ocp
make test-e2e-operator-ocp
```

## Manifests and Deployment

```bash
# Generate deployment manifests
make manifests

# Generate OLM bundle
make bundle

# Verify bundle is valid
make check-bundle
```

Manifests are generated in `build/_output/manifests/` from templates in `deploy/`. The operator.yaml is a template that gets populated with correct image references during `cluster-sync`.

## Vendoring

```bash
# Update vendor directory
make vendor
```

This tidy's both the main module and the `api/` module, then vendors dependencies. Go modules are used with `GOFLAGS=-mod=vendor` and `GOPROXY=direct`.

## Code Structure Notes

- Controllers use controller-runtime reconciliation pattern
- Handler filters events to only its node using labels.SelectorFromSet
- NetworkManager compatibility: >= 1.22 for versions > 0.15.0
- The handler requires a file lock (`pkg/file/lock.go`) to prevent concurrent nmstatectl operations
- Profiling can be enabled via ENABLE_PROFILER env var (default port 6060)

## CI Infrastructure

- Prow: https://prow.apps.ovirt.org/
- Flakefinder: https://storage.googleapis.com/kubevirt-prow/reports/flakefinder/nmstate/kubernetes-nmstate/index.html

## DCO Sign-off Required

All commits must include a DCO sign-off line. Use `git commit -s` or add manually:
```
Signed-off-by: Your Name <your.email@example.com>
```
