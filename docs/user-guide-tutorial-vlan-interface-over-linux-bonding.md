# Tutorial: Create a Bond with sub Vlan Interfaces and Connect it to a Node Interface

Use Node Network Configuration Policy to configure one new bond interface `bond0`
with slaves `eth1` and `eth2` and one vlan interface with `bond0` as parent interface

## Requirements

Before we start, please make sure that you have your Kubernetes/OpenShift
cluster ready. In order to do that, you can follow the guides of deployment on
[local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

## Configure bond with vlan

All you have to do in order to create the bond and the sub vlan interface on all nodes across cluster is
to apply the following policy:

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: bond0-vlan-eth1-eth2-policy
spec:
  desiredState:
    interfaces:
    - name: bond0
      type: bond
      state: up
      ipv4:
        address:
        - ip: 10.10.10.10
          prefix-length: 24
        enabled: true
      link-aggregation:
        mode: balance-rr
        options:
          miimon: '140'
        slaves:
        - eth1
        - eth2
    - name: bond0.102
      type: vlan
      state: up
      ipv4:
        address:
        - ip: 10.102.10.10
          prefix-length: 24
        enabled: true
      vlan:
        base-iface: bond0
        id: 102
EOF
```

You can also remove the bond and all is sub vlans with following:

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: bond0-vlan-eth1-eth2-policy
spec:
  desiredState:
    interfaces:
      - name: bond0
        type: bond
        state: absent
EOF
```
