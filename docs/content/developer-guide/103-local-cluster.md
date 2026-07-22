---
title: "Local Development Cluster"
weight: 30
type: docs
---

This page covers setting up and using a local development cluster for kubernetes-nmstate.

## Overview

The kubernetes-nmstate project uses [kubevirtci](https://github.com/kubevirt/kubevirtci) for local cluster development. This guide provides the necessary commands and configuration so you don't need to learn kubevirtci separately.

**Note**: kubevirtci is an external tool maintained by the KubeVirt project. Issues with kubevirtci itself are outside the scope of kubernetes-nmstate support.

For detailed information about deploying a local virtualized cluster, see the [local virtualized cluster guide]({{< relref "/deployment/local-cluster" >}}).

## Quick Reference

```bash
# Start local virtualized cluster (requires kubevirtci)
make cluster-up

# Deploy/update operator and handler to cluster
make cluster-sync

# Deploy only operator changes from rendered manifests
make cluster-sync-operator-manifests

# Stop cluster
make cluster-down

# Clean manifests-based cluster state
make clean-cluster-manifests
```

## Cluster Configuration

Configure the local cluster via environment variables:

- `KUBEVIRT_PROVIDER`: Kubernetes version (default: k8s-1.34)
- `KUBEVIRT_NUM_NODES`: Number of nodes (default: 3)
- `KUBEVIRT_NUM_SECONDARY_NICS`: Secondary NICs per node (default: 2)
- `KUBECONFIG`: Path to kubeconfig (auto-detected via ./cluster/kubeconfig.sh)
- `NMSTATE_VERSION`: When set to `latest`, uses nmstate-git from copr during `make cluster-up`
- `NM_VERSION`: When set to `latest`, installs NetworkManager from copr networkmanager/NetworkManager-main repository during `make cluster-up`

## Network Interface Names

Network interface names vary by provider:
- k8s providers: eth0, eth1, eth2
- Other providers: ens3, ens8, ens9

## Manifests

The operator `operator.yaml` manifest is rendered from the Helm chart template in `charts/kubernetes-nmstate/templates/` and gets populated with the correct docker image to use.

Every time `cluster-sync` is called, it re-renders the chart with the correct kubernetes-nmstate-handler image and applies the result.

Manifests are generated in `build/_output/manifests/` by rendering the Helm chart in `charts/kubernetes-nmstate/`; the rendered operator manifests live under `build/_output/manifests/kubernetes-nmstate/templates/`.

## Next Steps

Learn about code generation workflows: [Code Generation]({{< relref "/developer-guide/104-code-generation" >}})
