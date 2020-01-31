# Tutorial: configure VLAN on a Node Interface

Use Node Network Configuration Policy to configure a new vlan on a node interface `eth1`.

## Requirements

Before we start, please make sure that you have your Kubernetes/OpenShift
cluster ready. In order to do that, you can follow the guides of deployment on
[local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

## Configure vlan

All you have to do in order to configure the vlan on all nodes across cluster is
to apply the following policy
(In this example, the vlan ID is '102', the base interface is 'eth1' and the new vlan interface name is 'eth1.102'):

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: vlan-eth1-policy
spec:
  desiredState:
    interfaces:
    - name: eth1.102
      type: vlan
      state: up
      vlan:
        base-iface: eth1
        id: 102
EOF
```

You can also remove the vlan with following:

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: vlan-eth1 -policy
spec:
  desiredState:
    interfaces:
    - name: eth1.102
      type: vlan
      state: absent
EOF
```

## Selecting nodes

`NodeNetworkConfigurationPolicy` supports node selectors.
Thanks to them you can select a subset of nodes or a specific node by its name:

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: vlan-eth1-node01-policy
spec:
  nodeSelector:
    kubernetes.io/hostname: node01
  desiredState:
    interfaces:
    - name: eth1.102
      type: vlan
      state: up
      vlan:
        base-iface: eth1
        id: 102
EOF
```

## Set static IP address on a vlan

In order to set a static IP address on the vlan interface on all nodes across cluster, apply the following policy:
Note that when configuring a static IP you must use a node selector since we cannot set the same IP across multiple nodes.
```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: static-ip-on-vlan-policy
spec:
  nodeSelector:
    kubernetes.io/hostname: node01
  desiredState:
    interfaces:
    - name: eth1.102
      type: vlan
      state: up
      ipv4:
        address:
        - ip: 10.244.0.1
          prefix-length: 24
        dhcp: false
        enabled: true
EOF
```

## Set dynamic IP address on a vlan

In order to set dynamic IP address on the vlan interface on all nodes across cluster, apply the following policy:

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: dynamic-ip-on-vlan-policy
spec:
  desiredState:
    interfaces:
    - name: eth1.102
      type: vlan
      state: up
      ipv4:
        dhcp: true
        enabled: true
EOF
```
