---
title: "Deployment on Arbitrary Cluster"
permalink: /deployment-arbitrary-cluster/
---

In this guide, we will cover the installation of NetworkManager
and kubernetes-nmstate on your arbitrary cluster.

## Requirements

This guide requires you to have your own Kubernetes/OpenShift cluster. If you
don't have one and just want to try kubernetes-nmstate out, please refer to
the [deployment on local cluster](deployment-local-cluster.md) guide.

In order to get kubernetes-nmstate running, NetworkManager
must be installed on the node.

### NetworkManager

From [gnome](https://developer.gnome.org/NetworkManager/stable/NetworkManager.html):

"The NetworkManager daemon attempts to make networking configuration and
operation as painless and automatic as possible by managing the primary
network connection and other network interfaces, like Ethernet, Wi-Fi,
and Mobile Broadband devices. NetworkManager will connect any network device
when a connection for that device becomes available, unless that behavior
is disabled. Information about networking is exported via a D-Bus interface
to any interested application, providing a rich API with which to inspect
and control network settings and operation."

kubernetes-nmstate containers communicate with a NetworkManager instance running
on the node using D-Bus. Make sure that NetworkManager is installed and running
on each node.

```shell
yum install NetworkManager
systemctl start NetworkManager
```

## kubernetes-nmstate

Finally, we can install kubernetes-nmstate on our cluster. In order to do that,
please use [Cluster Network Addons Operator Project](https://github.com/kubevirt/cluster-network-addons-operator#nmstate).

You can stop here and play with the cluster on your own or continue with one of
the [user guides](../README.md#deployment-and-usage) that will guide you through
requesting node network states and configuring the nodes.
