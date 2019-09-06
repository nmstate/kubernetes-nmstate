# Tutorial: Create a Bond Interface and Connect it to a Node Interface

Use Node Network Configuration Policy to configure a new bond interface `bond0`
with slaves `eth1` and `eth2`.

## Requirements

Before we start, please make sure that you have your Kubernetes/OpenShift
cluster ready. In order to do that, you can follow the guides of deployment on
[local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

## Configure bond

All you have to do in order to create the bond on all nodes across cluster is
to apply the following policy:

```yaml
cat <<EOF | ./kubevirtci/cluster-up/kubectl.sh create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: bond0-eth1-eth2-policy
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
EOF
```

You can also remove the bond with following:

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: bond0-eth1-eth2-policy
spec:
  desiredState:
    interfaces:
      - name: bond0
        type: bond
        state: absent
EOF
```
