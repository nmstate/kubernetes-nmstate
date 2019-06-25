# Tutorial: Create an Linux Bridge and Connect it to a Node Interface

Use Node Network State to configure a new linux bridge `br1` connected
to node interface `eth1`. This bridge can be later used to connect pods to
L2 network using [linux-bridge-cni](https://github.com/containernetworking/plugins/tree/master/plugins/main/bridge).

## Requirements

Before we start, make sure that you have your Kubernetes/OpenShift cluster
ready. In order to do that, you can follow guides of deployment on
[local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

## Configure bridge

Install kubernetes-nmstate operator (if not already done).

Start the linux bridge br1 by patching desiredState at 'NodeNetworkState'
on the node we want to have it, in this case node01

```shell
# on local cluster
./kubevirtci/cluster-up/kubectl.sh patch --type merge nodenetworkstate node01 -p "$(cat docs/demos/create-br1-linux-bridge.yaml)"

# on arbitrary cluster
kubectl patch --type merge nodenetworkstate node01 -p "$(cat docs/demos/create-br1-linux-bridge.yaml)"
```

Delete the linux bridge br1 by patching again desiredState at NodeNetworkState
for node01 change bridge state to `absent`

```shell
# on local cluster
./kubevirtci/cluster-up/kubectl.sh patch --type merge nodenetworkstate node01 -p "$(cat docs/demos/delete-br1-linux-bridge.yaml)"

# on arbitrary cluster
kubectl patch --type merge nodenetworkstate node01 -p "$(cat docs/demos/delete-br1-linux-bridge.yaml")
```
