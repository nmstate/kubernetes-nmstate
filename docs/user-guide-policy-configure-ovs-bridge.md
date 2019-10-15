# Tutorial: Create a Open vSwitch Bridge and Connect it to a Node Interface

Use Node Network Configuration Policy to configure a new ovs bridge `br1` connected
to node interface `eth1`.

## Requirements

Before we start, please make sure that you have your Kubernetes/OpenShift
cluster ready. In order to do that, you can follow the guides of deployment on
[local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

You must also be sure that [OpenVSwitch](https://www.openvswitch.org/) and [NetworkManager OpenVSwitch plugin](https://developer.gnome.org/NetworkManager/stable/nm-openvswitch.html) are installed in the nodes.

Please also note that on OpenShift `openvswitch` is already installed as a daemon, so only `nm-openvswitch` is required.

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
        description: ovs bridge with eth1 as a port
        type: ovs-bridge
        state: up
        bridge:
          options:
            stp: false
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
        type: ovs-bridge
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
      - name: ovs-br0
        description: Ovs bridge with eth1 as a port
        type: ovs-bridge
        state: up
        bridge:
          options:
            stp: false
          port:
            - name: eth1
EOF
```
