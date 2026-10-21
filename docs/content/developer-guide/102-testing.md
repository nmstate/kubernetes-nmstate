---
title: "Testing"
weight: 20
type: docs
---

This page covers the various testing approaches for kubernetes-nmstate.

## Unit Tests

```bash
# Run all unit tests
make test/unit

# Run unit tests for specific package
make test/unit WHAT=./pkg/state/...

# Run API unit tests
make test/unit/api
```

Unit tests use Ginkgo v2 with parameters: `-r -keep-going --randomize-all --randomize-suites --race --trace`

## E2E Tests

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

### OKD installation CI

The `OKD upstream operator` GitHub Actions workflow runs on pull requests, pushes
to `main`, and manual dispatch. It provisions a single-node OKD cluster using
[CRC's OKD preset](https://crc.dev/docs/), then installs the upstream operator and
handler built from the checkout. CRC is pinned in `.github/workflows/okd.yml`;
its release selects the matching OKD bundle. No Red Hat pull secret or external
image registry credentials are required.

CRC is used because the pinned kubevirtci release no longer ships OKD/OCP
providers. CRC also [tests its OKD preset on GitHub-hosted Linux runners](https://github.com/crc-org/crc/blob/v2.64.0/.github/workflows/test-okd-bundle.yml).
The runner needs x86_64 Linux, KVM, libvirt, four CPUs, at least 16 GiB of host
memory (CRC uses 10.5 GiB), and sufficient free disk for the VM and image builds.
The workflow adjusts QEMU confinement only on its disposable hosted runner.

Images are built before the VM starts and pushed through a loopback port-forward
to OKD's internal registry using a short-lived, namespace-scoped service-account
token. The operator and handler service accounts are created first, so OpenShift
attaches their internal-registry pull secrets before any pod uses them.
Installation reuses the standard Helm deployment and readiness helpers.
The smoke test requires the operator Deployment, one ready handler pod, NMState
availability, and a successful NodeNetworkState update. It deliberately does not run the
multi-node/network-interface e2e suites on CRC's single-node cluster.

Cluster diagnostics and component logs are uploaded as `okd-installation`
artifacts, and the VM is deleted even on failure. CRC's startup banner, which
contains generated credentials, is discarded, and no credentials are uploaded.
To run the deployment-helper regression checks without a cluster:

```bash
bash hack/test-okd-smoke.sh
```

## Static Checks

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

## Next Steps

Learn how to set up a local development cluster: [Local Development Cluster]({{< relref "/developer-guide/103-local-cluster" >}})
