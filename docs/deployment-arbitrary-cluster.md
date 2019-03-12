# Deployment on Arbitrary Cluster

In this guide, we will cover the installation of Open vSwitch, NetworkManager
and kubernetes-nmstate on your arbitrary cluster.

## Requirements

This guide requires you to have your own Kubernetes/OpenShift cluster. If you
don't have one and just want to try kubernetes-nmstate out, please refer to
the [deployment on local cluster](deployment-local-cluster.md) guide.

In order to get kubernetes-nmstate running, NetworkManager, and optionally Open
vSwitch, must be installed on the node. This requirement will be dropped once we
make these components a part of the kubernetes-nmstate container images.

### NetworkManager

kubernetes-nmstate containers communicate with a NetworkManager instance running
on the node using D-Bus. Make sure that NetworkManager is installed and running
on each node.

```shell
yum install NetworkManager
systemctl start NetworkManager
```

### Open vSwitch

This part is optional. If you want to control Open vSwitch interfaces using
nmstate, Open vSwitch and its NetworkManager plugin must be installed on the
node.

#### Kubernetes

With Kubernetes, all you have to do is to install openvswitch package and its
NetworkManager plugin.

On CentOS (nmstate requires openvswitch >= 2.9.2. following commands will
install unofficial packages. Please note, that they are meant only for testing
and won't be automatically upgraded.):

```shell
yum install -y https://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-2.9.2-1.el7.x86_64.rpm https://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-devel-2.9.2-1.el7.x86_64.rpm https://cbs.centos.org/kojifiles/packages/dpdk/17.11/3.el7/x86_64/dpdk-17.11-3.el7.x86_64.rpm
yum install -y NetworkManager-ovs
systemctl daemon-reload
systemctl restart openvswitch NetworkManager
```

On Fedora:

```shell
dnf install -y openvswitch NetworkManager-ovs
systemctl daemon-reload
systemctl restart openvswitch NetworkManager
```

#### OpenShift

With OpenShift, the situation is a little bit challenging. If you just install
Open vSwitch on the node, it would break openshift-sdn networking. What you can
do instead, is to make the openshift-sdn Open vSwitch instance available on the
node.

In order to do that, install the following manifest. Please note that this
method is not officially supported and may break your OpenShift networking.

```shell
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/openshift-ovs-vsctl.yaml
```

Once Open vSwitch is exposed to the node, install the NetworkManager Open
vSwitch plugin.

```shell
yum install -y NetworkManager-ovs
systemctl daemon-reload
systemctl restart NetworkManager
```

## kubernetes-nmstate

Finally, we can install kubernetes-nmstate on our cluster.

Please note that this will not start kubernetes-nmstate controllers on the
cluster, only prepare it for their use. For more information, read
[active vs. on-demand guide](user-guide-active-vs-on-demand.md)

### Kubernetes

```shell
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/namespace.yaml
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/state-crd.yaml
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/configuration-policy-crd.yaml
```

### OpenShift

```shell
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/namespace.yaml
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/state-crd.yaml
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/configuration-policy-crd.yaml
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/openshift-scc.yaml
```

You can stop here and play with the cluster on your own or continue with one of
the [user guides](user-guide.md) that will guide you through requesting node
network states and configuring the nodes.
