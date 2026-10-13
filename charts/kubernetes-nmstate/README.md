# kubernetes-nmstate

[kubernetes-nmstate](https://github.com/nmstate/kubernetes-nmstate) provides
declarative node network configuration driven through the Kubernetes API. It
uses [nmstate](https://nmstate.io) to configure networking on the cluster
nodes and to report their current state.

NetworkManager must be running on the nodes, it is the only external
dependency.

## Prerequisites

- Helm >= 3.8 (OCI registry support)
- NetworkManager running on the nodes

## Install

```shell
helm install nmstate oci://quay.io/nmstate/kubernetes-nmstate \
  --version <version> \
  --namespace nmstate \
  --create-namespace
```

This deploys the operator and, by default (`nmstate.enabled=true`), an
`NMState` custom resource that makes the operator deploy the
kubernetes-nmstate handler on all nodes.

kubernetes-nmstate is a cluster singleton: only one release of this chart per
cluster is supported. The chart creates cluster scoped resources with fixed
names (such as the `nmstate-operator` ClusterRole and ClusterRoleBinding), so
installing a second release would conflict with the first one.

## Values

| Key | Default | Description |
|-----|---------|-------------|
| `operator.image` | `""` | Operator image; empty means `quay.io/nmstate/kubernetes-nmstate-operator:<appVersion>` |
| `operator.pullPolicy` | `IfNotPresent` | Operator image pull policy |
| `handler.image` | `""` | Handler image; empty means `quay.io/nmstate/kubernetes-nmstate-handler:<appVersion>` |
| `handler.pullPolicy` | `IfNotPresent` | Handler image pull policy |
| `handler.namespace` | `nmstate` | Namespace the operator deploys the handler into |
| `handler.prefix` | `""` | Optional name prefix for the handler resources deployed by the operator; when empty no `HANDLER_PREFIX` env is rendered |
| `handler.imageEnvVar` | `RELATED_IMAGE_HANDLER_IMAGE` | Name of the operator env var carrying the handler image |
| `plugin.image` | `""` | Console plugin image for downstream distributions; when empty no `PLUGIN_IMAGE` env is rendered |
| `monitoring.namespace` | `monitoring` | Cluster monitoring namespace |
| `createNamespace` | `false` | Emit a Namespace object for the release namespace (use `helm install --create-namespace` instead) |
| `nmstate.enabled` | `true` | Create the `NMState` custom resource (named `nmstate`) at install time |
| `nmstate.spec` | `{}` | Passthrough for `NMState` spec fields (`nodeSelector`, `tolerations`, ...) |

## Usage

Configure node networking by creating a `NodeNetworkConfigurationPolicy`:

```yaml
apiVersion: nmstate.io/v1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: linux-bridge
spec:
  desiredState:
    interfaces:
    - name: br1
      description: Linux bridge with eth1 as a port
      type: linux-bridge
      state: up
      ipv4:
        dhcp: true
        enabled: true
      bridge:
        options:
          stp:
            enabled: false
        port:
        - name: eth1
```

The current state of each node is reported through `NodeNetworkState`
objects, and the result of applying a policy through
`NodeNetworkConfigurationEnactment` objects.

## Upgrade

Helm does not modify custom resource definitions shipped in the chart's
`crds/` directory on upgrade, so apply the NMState CRD manually first:

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

## Documentation

- [Deployment guide](https://nmstate.github.io/kubernetes-nmstate/deployment/helm)
- [User guide](https://nmstate.github.io/kubernetes-nmstate/user-guide/)
- [Examples](https://github.com/nmstate/kubernetes-nmstate/tree/main/docs/examples)
