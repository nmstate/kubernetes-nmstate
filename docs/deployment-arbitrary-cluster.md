# Deployment on Arbitrary Cluster

In this guide, we will cover the installation of etworkManager
and kubernetes-nmstate on your arbitrary cluster.

## Requirements

This guide requires you to have your own Kubernetes/OpenShift cluster. If you
don't have one and just want to try kubernetes-nmstate out, please refer to
the [deployment on local cluster](deployment-local-cluster.md) guide.

In order to get kubernetes-nmstate running, NetworkManager
must be installed on the node.

### NetworkManager

kubernetes-nmstate containers communicate with a NetworkManager instance running
on the node using D-Bus. Make sure that NetworkManager is installed and running
on each node.

```shell
yum install NetworkManager
systemctl start NetworkManager
```

## kubernetes-nmstate

Finally, we can install kubernetes-nmstate on our cluster.

### Kubernetes

```shell
# Install k8s resources and kubernets-nmstate operator
for manifest in "service_account.yaml role.yaml role_binding.yaml crds/nmstate_v1_nodenetworkstate_crd.yaml operator.yaml"; do
    kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/deploy/0.0.3/$manifest
done
```

### OpenShift

```shell
# Install k8s resources , openshift scc and kubernets-nmstate operator
for manifest in "service_account.yaml role.yaml role_binding.yaml openshift/scc.yaml crds/nmstate_v1_nodenetworkstate_crd.yaml operator.yaml"; do
    kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/deploy/0.0.3/$manifest
done
```

You can stop here and play with the cluster on your own or continue with one of
the [user guides](user-guide.md) that will guide you through requesting node
network states and configuring the nodes.
