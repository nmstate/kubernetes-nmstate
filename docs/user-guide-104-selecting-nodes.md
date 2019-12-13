# 104: Selecting Nodes

TODO

## Requirements

Before we start, please make sure that you have your Kubernetes/OpenShift
cluster ready. In order to do that, you can follow the guides of deployment on
[local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

## Selecting nodes

`NodeNetworkConfigurationPolicy` supports node selectors. Thanks to them you can
select a subset of nodes or a specific node by its name:

```yaml
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: eth1-on-node01-is-up
spec:
  nodeSelector:
    kubernetes.io/hostname: node01
  desiredState:
    interfaces:
    - name: eth1
      type: ethernet
      state: up
EOF
```

```
TODO output
```

When multiple node selectors are specified, all of them have to apply.

## Check which nodes were matching

TODO You can look into `NodeNetworkConfigurationEnactment` objects of given policy to see which nodes matched the selector.

```
kubectl get nodenetworkconfigurationenactments -l nmstate.io/policy=br1-eth1
```

```
TODO example output with NonMatching and Available shown there
```

`Matching` condition of the object would tell you. If it is not matching, it includes list of labels that did not match.

```
kubectl get nodenetworkconfigurationenactment node01.br1-eth1 -o jsonpath='{.status.conditions[?(@.type=="Matching")].message}'
```

```
TODO: output of matching
```
