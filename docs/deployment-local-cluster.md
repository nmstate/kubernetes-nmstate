# Deployment on Local Cluster

Kubernetes-nmstate project allows you to spin up a virtualized
Kubernets/OpenShift cluster thanks to
[kubevirtci](https://github.com/kubevirt/kubevirtci) project.
In this guide, we will create a local Kubernetes
cluster with two nodes and preinstalled node dependencies. Then we will deploy
kubernetes-nmstate from local sources.

If you want to deploy kubernetes-nmstate on your arbitrary cluster, read
the [deployment on arbitrary cluster guide](deployment-arbitrary-cluster.md).

Start your local cluster. If you want to use OpenShift instead of Kubernetes or
a different amount of nodes, check the
[development guide](developer-guide.md#local-cluster).

Start the local cluster

```shell
make cluster-up
```

Stop the local cluster

```shell

make cluster-down
```

Build kubernetes-nmstate from local sources and install it on the cluster.

```shell
make cluster-sync
```

You can ssh into the created nodes using `kubevirtci/clusetr-up/ssh.sh`.

```shell
kubevirtci/cluster-up/ssh.sh node01
```

Finally, you can access Kubernetes API using `kubevirtci/cluster-up/kubectl.sh`.

```shell
kubevirtci/cluster-up/kubectl.sh get nodes
```

If you want to use `kubectl` to access the cluster, start a proxy.

```shell
kubevirtci/cluster-up/kubectl.sh proxy --port=8080 --disable-filter=true &
```

You can stop here and play with the cluster on your own or continue with the other
[user guides](user-guide.md) that will guide you through requesting of node
interfaces and their configuration.
