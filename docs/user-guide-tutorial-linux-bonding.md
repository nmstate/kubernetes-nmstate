# Tutorial: Linux Bonding

Use Node Network Configuration Policy to configure a new bond interface `bond0`
with slaves `eth1` and `eth2`.

All you have to do in order to create the bond on all nodes across cluster is
to apply the following policy:

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: bond0-eth1-eth2
spec:
  desiredState:
    interfaces:
    - name: bond0
      type: bond
      state: up
      ipv4:
        dhcp: true
        enabled: true
      link-aggregation:
        mode: balance-rr
        options:
          miimon: '140'
        slaves:
        - eth1
        - eth2
```

You can also remove the bond with following:

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: bond0-eth1-eth2
spec:
  desiredState:
    interfaces:
      - name: bond0
        type: bond
        state: absent
```
