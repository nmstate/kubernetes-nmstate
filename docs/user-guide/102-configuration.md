---
title: "Configuration"
tag: "user-guide"
---

The operator allows users to configure various network interface types, DNS and
routing on cluster nodes. The configuration is driven by two main object types,
`NodeNetworkConfigurationPolicy` (Policy) and
`NodeNetworkConfigurationEnactment` (Enactment).

A Policy describes what is the desired network configuration on cluster nodes.
It is created by users and applies cluster-wide.
An Enactment, on the other hand, is a read-only object that carries execution
state of a Policy per each Node.

Multiple policies can be created at a time, but users need to avoid
inter-policy dependencies. If two policies need to be applied in specific
order, they should form a single policy.

Policies are applied on a node via NetworkManager. Thanks to this, the
configuration is persistent on the node and survives reboots.

:warning: Changing a policy that is in progress has undefined behaviour.

## Creating interfaces

Each Policy has a name (`metadata.name`) and desired state
(`spec.desiredState`). The desired state can contains declarative specification
of the node network configuration following [nmstate
API](https://nmstate.github.io/).

In the following example, we will create a Policy that configures a bonding over
NICs `eth1` and `eth2` across the whole cluster.

First of all, let's apply the manifest:

<!-- When updating following example, don't forget to update respective attached file -->

[Download example]({{ "user-guide/bond0-eth1-eth2_up.yaml" | relative_url }})

```yaml
{% include_absolute 'user-guide/bond0-eth1-eth2_up.yaml' %}
```

```shell
kubectl apply -f bond0-eth1-eth2_up.yaml
```

Wait for the Policy to be successfully applied:

```shell
kubectl wait nncp bond0-eth1-eth2 --for condition=Available --timeout 2m
```

List all the Policies applied on the cluster:

```shell
kubectl get nodenetworkconfigurationpolicies
```

```
NAME              STATUS
bond0-eth1-eth2   SuccessfullyConfigured
```

We can also use short name `nncp` to reach the same effect:

```shell
kubectl get nncp
```

```
NAME              STATUS
bond0-eth1-eth2   SuccessfullyConfigured
```

By using `-o yaml` we obtain the full Policy with its current state:

```shell
kubectl get nncp bond0-eth1-eth2 -o yaml
```

```yaml
# output truncated
status:
  conditions:
  - lastHearbeatTime: "2020-02-07T10:27:09Z"
    lastTransitionTime: "2020-02-07T10:27:09Z"
    message: 2/2 nodes successfully configured
    reason: SuccessfullyConfigured
    status: "True"
    type: Available
  - lastHearbeatTime: "2020-02-07T10:27:09Z"
    lastTransitionTime: "2020-02-07T10:27:09Z"
    reason: SuccessfullyConfigured
    status: "False"
    type: Degraded
```

The `status` section contains conditions (previously used with the `wait`
command). There is one for successful configuration (`Available`) and one for a
failed one (`Degraded`). As the output show, the Policy was applied successfully
on both nodes.

As mentioned in the introduction, for each Policy there is a set of Enactments
created by the operator. List of Enactments of all Policies and Nodes:

```shell
kubectl get nodenetworkconfigurationenactments
```

```
NAME                     STATUS
node01.bond0-eth1-eth2   SuccessfullyConfigured
node02.bond0-eth1-eth2   SuccessfullyConfigured
```

We can also use short name `nnce` to reach the same effect:

```shell
kubectl get nnce
```

```
NAME                     STATUS
node01.bond0-eth1-eth2   SuccessfullyConfigured
node02.bond0-eth1-eth2   SuccessfullyConfigured
```

By using `-o yaml` we obtain the full status of given Enactment:

```shell
kubectl get nnce node01.bond0-eth1-eth2 -o yaml
```

```yaml
# output truncated
status:
  conditions:
  - lastHearbeatTime: "2020-02-07T10:27:09Z"
    lastTransitionTime: "2020-02-07T10:27:09Z"
    reason: SuccessfullyConfigured
    status: "False"
    type: Failing
  - lastHearbeatTime: "2020-02-07T10:27:09Z"
    lastTransitionTime: "2020-02-07T10:27:09Z"
    message: successfully reconciled
    reason: SuccessfullyConfigured
    status: "True"
    type: Available
  - lastHearbeatTime: "2020-02-07T10:27:09Z"
    lastTransitionTime: "2020-02-07T10:27:09Z"
    reason: SuccessfullyConfigured
    status: "False"
    type: Progressing
  desiredState:
    interfaces:
    - ipv4:
        dhcp: true
        enabled: true
      link-aggregation:
        mode: balance-rr
        slaves:
        - eth1
        - eth2
      name: bond0
      state: up
      type: bond
```

The output contains the `desiredState` applied by the Policy for the given Node.
It also contains a list of conditions. This list is more detailed than the one
in Policy. It shows if the `desiredState` is currently being applied on
the node (`Progressing`), if the configuration failed (`Failing`) or succeeded
(`Available`).

<!-- TODO: Once we have an article about node selectors, link it here -->

Our Enactment shows that it successfully applied the configuration, let's use
`NodeNetworkState` to verify it:

```shell
kubectl get nns node01 -o yaml
```

```yaml
# Output truncated
status:
  currentState:
    interfaces:
    - ipv4:
        address:
        - ip: 192.168.66.127
          prefix-length: 24
        auto-dns: true
        auto-gateway: true
        auto-routes: true
        dhcp: true
        enabled: true
      ipv6:
        autoconf: false
        dhcp: false
        enabled: false
      link-aggregation:
        mode: balance-rr
        options: {}
        slaves:
        - eth2
        - eth1
      mac-address: 52:55:00:D1:56:01
      mtu: 1500
      name: bond0
      state: up
      type: bond
    - ipv4:
        dhcp: false
        enabled: false
      ipv6:
        autoconf: false
        dhcp: false
        enabled: false
      mac-address: 52:55:00:D1:56:01
      mtu: 1500
      name: eth1
      state: up
      type: ethernet
    - ipv4:
        dhcp: false
        enabled: false
      ipv6:
        autoconf: false
        dhcp: false
        enabled: false
      mac-address: 52:55:00:D1:56:01
      mtu: 1500
      name: eth2
      state: up
      type: ethernet
```

As seen in the output, the configuration is indeed applied and there is a bond
available with two NICs used as its ports.

## Removing interfaces

One may expect that by removal of the Policy, the applied configuration would be
reverted. However, that's not the case. The Policy is not owning the
configuration on the host, it is merely applying the difference needed to reach
the desired state. After removal of the Policy, the configuration on the node
remains the same.

In order to remove a configured interface from nodes, we need to explicitly
specify it in the Policy. That can by done by changing the `state: up` of the
interface to `state: absent`:

<!-- When updating following example, don't forget to update respective attached file -->

[Download example]({{ "user-guide/bond0-eth1-eth2_absent.yaml" | relative_url }})

```yaml
{% include_absolute 'user-guide/bond0-eth1-eth2_absent.yaml' %}
```

```shell
kubectl apply -f bond0-eth1-eth2_absent.yaml
```

> **Note:** Alternative approach to `apply` would be to use `edit` and change the state from `up` to `absent` manually:
>
> ```shell
> kubectl edit nncp bond0-eth1-eth2
> ```

Wait for the Policy to be applied and the interface removed:

```shell
kubectl wait nncp bond0-eth1-eth2 --for condition=Available --timeout 2m
```

After the Policy is applied, it is not needed anymore and it can be deleted:

```shell
kubectl delete nncp bond0-eth1-eth2
```

## Restore original configuration

Another maybe surprising behavior is, that by removing an interface, original
configuration of the node interfaces is not restored. In case of the bonding it
means that after it is deleted, its ports won't come back up as standalone
ports, even if they had previously configured IP address. The operator is not
owning the interfaces and does not want to do anything that is not explicitly
specified, that's up to the user.

`NodeNetworkState` shows that both of the NICs are now down and without any IP
configuration.

```shell
kubectl get nns node01 -o yaml
```

```yaml
# output truncated
status:
    interfaces:
    - ipv4:
        enabled: false
      ipv6:
        enabled: false
      mac-address: 52:55:00:D1:56:00
      mtu: 1500
      name: eth1
      state: down
      type: ethernet
    - ipv4:
        enabled: false
      ipv6:
        enabled: false
      mac-address: 52:55:00:D1:56:01
      mtu: 1500
      name: eth2
      state: down
      type: ethernet
```

In order to configure IP on previously attached NICs, apply a new Policy:

<!-- When updating following example, don't forget to update respective attached file -->

[Download example]({{ "/user-guide/eth1-eth2_up.yaml" | relative_url }})

```yaml
{% include_absolute 'user-guide/eth1-eth2_up.yaml' %}
```

```shell
kubectl apply -f eth1-eth2_up.yaml
```

Wait for the Policy to get applied:

```shell
kubectl wait nncp eth1 eth2 --for condition=Available --timeout 2m
```

Both NICs are now back up and with assigned IPs:

```shell
kubectl get nns node01 -o yaml
```

```yaml
# output truncated
status:
  currentState:
    interfaces:
    - ipv4:
        address:
        - ip: 192.168.66.126
          prefix-length: 24
        auto-dns: true
        auto-gateway: true
        auto-routes: true
        dhcp: true
        enabled: true
      ipv6:
        autoconf: false
        dhcp: false
        enabled: false
      mac-address: 52:55:00:D1:56:00
      mtu: 1500
      name: eth1
      state: up
      type: ethernet
    - ipv4:
        address:
        - ip: 192.168.66.127
          prefix-length: 24
        auto-dns: true
        auto-gateway: true
        auto-routes: true
        dhcp: true
        enabled: true
      ipv6:
        autoconf: false
        dhcp: false
        enabled: false
      mac-address: 52:55:00:D1:56:01
      mtu: 1500
      name: eth2
      state: up
      type: ethernet
```

## Selecting nodes

All the Policies we used so far were applied on all nodes across the cluster. It
is however possible to select only a subset of nodes using via a node selector.

In the following example, we configure a VLAN interface with tag 100 over a NIC
`eth1`. This configuration will be done only on node which has labels matching
all the key-value pairs in the `nodeSelector`:

<!-- When updating following example, don't forget to update respective attached file -->

[Download example]({{ "user-guide/vlan100_node01_up.yaml" | relative_url }})

```yaml
{% include_absolute 'user-guide/vlan100_node01_up.yaml %}
```

```shell
kubectl apply -f vlan100_node01_up.yaml
```

Wait for the Policy to get applied:

```shell
kubectl wait nncp vlan100 --for condition=Available --timeout 2m
```

The list of Enactments then shows that the Policy has been applied only on
Node `node01`.

```shell
kubectl get nnce
```

```
NAME             STATUS
node01.eth1      SuccessfullyConfigured
node01.eth2      SuccessfullyConfigured
node01.vlan100   SuccessfullyConfigured
node02.eth1      SuccessfullyConfigured
node02.eth2      SuccessfullyConfigured
```

## Configuring multiple nodes concurrently

By default, Policy configuration is applied sequentially, one node at a time.
This configuration strategy is safe and prevents the entire cluster from being
temporarily unavailable, if the applied configuration breaks network connectivity.

For big clusters however, it may take too much time for a configuration to finish.
In such a case, `maxUnavailable` can be used to define portion size of a cluster
that can apply a policy configuration concurrently.
MaxUnavailable specifies percentage or a constant number of nodes that
can be progressing a policy at a time. The default is "50%" of cluster nodes.

The following policy specifies that up to 3 nodes may be progressing concurrently:

```yaml
{% include_absolute 'user-guide/linux-bridge_maxunavailable.yaml %}
```

```shell
kubectl apply -f linux-bridge_maxunavailable.yaml
```

```
NAME                                 STATUS
node01.linux-bridge-maxunavailable   ConfigurationProgressing
node02.linux-bridge-maxunavailable   ConfigurationProgressing
node03.linux-bridge-maxunavailable   SuccessfullyConfigured
node04.linux-bridge-maxunavailable   ConfigurationProgressing
node05.linux-bridge-maxunavailable   ConfigurationProgressing
node06.linux-bridge-maxunavailable   ConfigurationProgressing
```

## Continue reading

The following tutorial will guide you through troubleshooting of a failed
configuration:
[Troubleshooting]({{ "user-guide/103-troubleshooting.html" | relative_url }})
