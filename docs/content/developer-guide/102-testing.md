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
