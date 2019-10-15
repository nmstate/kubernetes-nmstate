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

By doing this though, we will be able to create an Open vSwitch bridge on the host but the host may loose connectivity since its nic is now connected to the bridge.
In order to have the host accessible, we need to provide the bridge an ip address. This is achieved by using an Open vSwitch internal interface.

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: br1-eth1-policy
spec:
  desiredState:
    interfaces:
      - name: ovs0
        type: ovs-interface
        state: up
        ipv4:
          enabled: true
          address:
            - ip: 192.0.2.1
              prefix-length: 24
      - name: br1
        description: ovs bridge with eth1 as a port and ovs0 as an internal interface
        type: ovs-bridge
        state: up
        bridge:
          options:
            stp: true
          port:
            - name: eth1
              type: system
            - name: ovs0
              type: internal
EOF
```



You can also remove the bridge with the following command:

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
