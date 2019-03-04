# Deployment on Local Cluster

Cluster Network Addons Operators project allows you to spin up a virtualized
Kubernets/OpenShift cluster. In this guide, we will create a local Kubernetes
cluster with two nodes and preinstalled node dependencies. Then we will deploy
kubernetes-nmstate from local sources.

If you want to deploy kubernetes-nmstate on your arbitrary cluster, read
the [deployment on arbitrary cluster guide](deployment-arbitrary-cluster.md).

Start your local cluster. If you want to use OpenShift instead of Kubernetes or
a different amount of nodes, check the
[development guide](developer-guide.md#local-cluster).

```shell
KUBEVIRT_NUM_NODES=2 make cluster-up
```

Build kubernetes-nmstate from local sources and install it on the cluster.
(Please note that this will not start kubernetes-nmstate controllers on the
cluster, for more information, read the
[active vs. on-demand user guide](user-guide-active-vs-on-demand.md))

```shell
make cluster-sync
```

You can ssh into the created nodes using `cluster/cli.sh`.

```shell
cluster/cli.sh ssh node01
```

Finally, you can access Kubernetes API using `cluster/kubectl.sh`.

```shell
./cluster/kubectl.sh get nodes
```

If you want to use `kubectl` to access the cluster, start a proxy.

```shell
./cluster/kubectl.sh proxy --port=8080 --disable-filter=true &
```

You can stop here and play with the cluster on your own or continue with the other
[user guides](user-guide.md) that will guide you through requesting of node
interfaces and their configuration.
