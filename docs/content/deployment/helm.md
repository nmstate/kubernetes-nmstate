---
title: "Deployment with Helm"
weight: 30
type: docs
---

kubernetes-nmstate can be installed with [Helm](https://helm.sh) (>= 3.8)
from an OCI registry.

kubernetes-nmstate is a cluster singleton: only one release of this chart per
cluster is supported. The chart creates cluster-scoped resources with fixed
names (such as the `nmstate-operator` ClusterRole and ClusterRoleBinding), so
installing a second release would conflict with the first one.

## Install

```shell
helm install nmstate oci://quay.io/nmstate/kubernetes-nmstate \
  --version <version> \
  --namespace nmstate \
  --create-namespace
```

This deploys the operator and, by default (`nmstate.enabled=true`), an
`NMState` custom resource that makes the operator deploy the
kubernetes-nmstate handler on all nodes. NetworkManager must be running on
the nodes (see the
[arbitrary cluster guide]({{< relref "/deployment/arbitrary-cluster" >}})).

Note: if your cluster enforces Pod Security admission and the handler
namespace equals the release namespace, the operator labels the namespace
for privileged workloads automatically.

## Values

| Key | Default | Description |
|-----|---------|-------------|
| `operator.image` | `""` | Operator image; empty means `quay.io/nmstate/kubernetes-nmstate-operator:<appVersion>` |
| `operator.pullPolicy` | `IfNotPresent` | Operator image pull policy |
| `handler.image` | `""` | Handler image; empty means `quay.io/nmstate/kubernetes-nmstate-handler:<appVersion>` |
| `handler.pullPolicy` | `IfNotPresent` | Handler image pull policy |
| `handler.namespace` | `nmstate` | Namespace the operator deploys the handler into |
| `monitoring.namespace` | `monitoring` | Cluster monitoring namespace |
| `createNamespace` | `false` | Emit a Namespace object for the release namespace (use `helm install --create-namespace` instead) |
| `nmstate.enabled` | `true` | Create the `NMState` custom resource (named `nmstate`) at install time |
| `nmstate.spec` | `{}` | Passthrough for `NMState` spec fields (`nodeSelector`, `tolerations`, ...) |

## Upgrade

Helm does not modify custom resource definitions shipped in the chart's
`crds/` directory on upgrade. Apply the NMState CRD manually first:

```shell
kubectl apply -f https://github.com/nmstate/kubernetes-nmstate/releases/download/<version>/nmstate.io_nmstates.yaml
helm upgrade nmstate oci://quay.io/nmstate/kubernetes-nmstate \
  --version <version> \
  --namespace nmstate
```

## Uninstall

```shell
helm uninstall nmstate --namespace nmstate --wait
```
