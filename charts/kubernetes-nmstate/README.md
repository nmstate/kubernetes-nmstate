# kubernetes-nmstate

kubernetes-nmstate is a Kubernetes operator for declarative node network
configuration using [nmstate](https://nmstate.io/) and NetworkManager. It
reports the current network state of cluster nodes and applies desired
configuration through Kubernetes custom resources.

## Prerequisites

Install [Helm](https://helm.sh) (>= 3.8) and ensure NetworkManager is running
on the cluster nodes. See the
[arbitrary cluster guide](https://nmstate.github.io/kubernetes-nmstate/deployment/arbitrary-cluster/)
for node requirements.

kubernetes-nmstate is a cluster singleton: only one release of this chart per
cluster is supported. The chart creates cluster-scoped resources with fixed
names (such as the `nmstate-operator` ClusterRole and ClusterRoleBinding), so
installing a second release would conflict with the first one.

## Install

Choose a chart version from
[Artifact Hub](https://artifacthub.io/packages/helm/kubernetes-nmstate/kubernetes-nmstate).
Use that version without a leading `v` for `<version>` in the commands below.

```shell
helm install nmstate oci://quay.io/nmstate/kubernetes-nmstate \
  --version <version> \
  --namespace nmstate \
  --create-namespace
```

This deploys the operator and, by default (`nmstate.enabled=true`), an
`NMState` custom resource that makes the operator deploy the
kubernetes-nmstate handler on all nodes.

If your cluster enforces Pod Security admission and the handler namespace
equals the release namespace, the operator labels the namespace for
privileged workloads automatically.

### Verify a signed release

For signed chart versions, download the public key linked in the package's
signing information on Artifact Hub as `nmstate-helm-signing.asc`. Convert
it to a binary keyring and verify the chart when downloading it:

```shell
gpg --dearmor --output nmstate-helm-keyring.gpg nmstate-helm-signing.asc
helm pull oci://quay.io/nmstate/kubernetes-nmstate \
  --version <version> \
  --verify \
  --keyring ./nmstate-helm-keyring.gpg
```

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

## Upgrade

Helm does not modify custom resource definitions shipped in the chart's
`crds/` directory on upgrade. Apply the NMState CRD manually first. GitHub
release tags use a leading `v`, unlike Helm chart versions:

```shell
kubectl apply -f https://github.com/nmstate/kubernetes-nmstate/releases/download/v<version>/nmstate.io_nmstates.yaml
helm upgrade nmstate oci://quay.io/nmstate/kubernetes-nmstate \
  --version <version> \
  --namespace nmstate
```

## Uninstall

```shell
helm uninstall nmstate --namespace nmstate --wait
```

## Usage and support

- [Reporting node network state](https://nmstate.github.io/kubernetes-nmstate/user-guide/101-reporting/)
- [Configuring node networks](https://nmstate.github.io/kubernetes-nmstate/user-guide/102-configuration/)
- [Troubleshooting](https://nmstate.github.io/kubernetes-nmstate/user-guide/103-troubleshooting/)
- [Report an issue](https://github.com/nmstate/kubernetes-nmstate/issues)
