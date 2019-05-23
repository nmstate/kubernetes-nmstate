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

Install kubernetes-nmstate state handler in active mode (if not already done).

```shell
# on local cluster
./cluster/kubectl.sh create -f _out/manifests/state-controller-ds.yaml

# on arbitrary cluster
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/state-controller-ds.yaml
```

Start the linux bridge br1 by creating the 'NodeNetworkState' 
with it on 'up' state

```shell
# on local cluster
./cluster/kubectl.sh create -f _out/manifests/create-br1-linux-bridge.yaml

# on arbitrary cluster
kubectl create -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/create-br1-linux-bridge.yaml
```

Delete the linux bridge br1 by updating the 'NodeNetworkState' 
with it on 'absent' state

```shell
# on local cluster
./cluster/kubectl.sh apply -f _out/manifests/delete-br1-linux-bridge.yaml

# on arbitrary cluster
kubectl create -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/delete-br1-linux-bridge.yaml
```
