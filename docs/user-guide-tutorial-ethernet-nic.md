# Tutorial: Ethernet NIC

TODO Use Node Network Configuration Policy to configure a new bond interface `bond0`
with slaves `eth1` and `eth2`.

TODO motivation why to do such a config

TODO All you have to do in order to create the bond on all nodes across cluster is
to apply the following policy:

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: default-nic
spec:
  desiredState:
    interfaces:
    - name: eth0
      type: ethernet
      state: up
      ipv4:
        dhcp: true
        enabled: true
```

TODO You can also remove the bond with following:
TODO try to drop the type.
TODO note that it may break the connectivity to the node. in such a case, rollback to the original state will happen.

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: default-nic
spec:
  desiredState:
    interfaces:
    - name: eth0
      type: ethernet
      state: absent
```
