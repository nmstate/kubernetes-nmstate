---
title: "Code Generation"
weight: 40
type: docs
---

This page covers code generation and manifest workflows for kubernetes-nmstate.

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

**Tip**: When using Claude Code, you can run `/review-api-changes` to automatically review your API changes for compatibility and best practices.

## Manifests and Deployment

**Note**: This section covers the [Operator Lifecycle Manager (OLM)](https://olm.operatorframework.io/) framework for operator distribution.

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

This tidies both the main module and the `api/` module, then vendors dependencies. Go modules are used with `GOFLAGS=-mod=vendor` and `GOPROXY=direct`.

## Next Steps

For advanced topics, see: [Advanced Topics]({{< relref "/developer-guide/105-advanced" >}})
