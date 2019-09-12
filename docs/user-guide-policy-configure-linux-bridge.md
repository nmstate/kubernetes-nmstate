# Tutorial: Create a Linux Bridge and Connect it to a Node Interface

Use Node Network Configuration Policy to configure a new linux bridge `br1` connected
to node interface `eth1`.

## Requirements

Before we start, please make sure that you have your Kubernetes/OpenShift
cluster ready. In order to do that, you can follow the guides of deployment on
[local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

## Configure bridge

All you have to do in order to create the bridge on all nodes across cluster is
to apply the following policy:

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: br1-eth1-policy
spec:
  desiredState:
    interfaces:
      - name: br1
        description: Linux bridge with eth1 as a port
        type: linux-bridge
        state: up
        bridge:
          options:
            stp:
              enabled: false
          port:
            - name: eth1
EOF
```

You can also remove the bridge with following:

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: br1-eth1-policy
spec:
  desiredState:
    interfaces:
      - name: br1
        type: linux-bridge
        state: absent
EOF
```

## Selecting nodes

`NodeNetworkConfigurationPolicy` supports node selectors. Thanks to them you can
select a subset of nodes or a specific node by its name:

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: br1-eth1-policy
spec:
  nodeSelector:
    kubernetes.io/hostname: node01
  desiredState:
    interfaces:
      - name: br1
        description: Linux bridge with eth1 as a port
        type: linux-bridge
        state: up
        bridge:
          options:
            stp:
              enabled: false
          port:
            - name: eth1
              stp-hairpin-mode: false
              stp-path-cost: 100
              stp-priority: 32
EOF
```
