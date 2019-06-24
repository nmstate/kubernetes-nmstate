# Tutorial: Reporting State

In this example, we will use kubernetes-nmstate to report state of network
interfaces on our cluster nodes.

## Requirements

Before we start, make sure that you have your Kubernetes/OpenShift cluster
ready with OVS. In order to do that, you can follow guides of deployment on
[local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

## Periodically report state from all nodes

Install kubernetes-nmstate operator (if not done yet). This
operator will periodically update reported state of node interfaces. It will
also apply desired specification of node networking if it is changed.

Read Node Network States from all nodes.

```shell
# on local cluster
./cluster/kubectl.sh get nodenetworkstates -o yaml

# on arbitrary cluster
kubectl get nodenetworkstates -o yaml
```

Then, read reported network state from selected node.

```shell
# on local cluster
./cluster/kubectl.sh get nodenetworkstate node01 -o yaml

# on arbitrary cluster
kubectl e node01 -o yaml
```
