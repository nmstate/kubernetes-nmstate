# Introduction: Configuration

TODO: what are policies and enactments

TODO: desired state API

## Creating interfaces

<!-- When updating following example, don't forget to update respective attached file -->

[Download example](example_bond0-eth1-eth2.yaml)

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: bond0-eth1-eth2
spec:
  desiredState:
    interfaces:
    - name: bond0
      type: bond
      state: up
      ipv4:
        dhcp: true
        enabled: true
      link-aggregation:
        mode: balance-rr
        slaves:
        - eth1
        - eth2
```

```shell
kubectl apply -f example_bond0-eth1-eth2.yaml
```

```shell
kubectl wait nncp bond0-eth1-eth2 --for condition=Available
```

```shell
kubectl get nodenetworkconfigurationpolicies
```

```shell
kubectl get nncp
```

```
NAME               STATUS
bond0-eth1-eth2   SuccessfullyConfigured
```

```shell
kubectl get nncp bond0-eth1-eth2 -o yaml
```

```yaml
# Output truncated
status:
  conditions:
  - lastHearbeatTime: "2020-01-31T16:15:58Z"
    lastTransitionTime: "2020-01-31T16:15:57Z"
    message: 2/2 nodes successfully configured
    reason: SuccessfullyConfigured
    status: "True"
    type: Available
  - lastHearbeatTime: "2020-01-31T16:15:58Z"
    lastTransitionTime: "2020-01-31T16:15:57Z"
    reason: SuccessfullyConfigured
    status: "False"
    type: Degraded
```

You see it is applied to two nodes.

```shell
kubectl get nodenetworkconfigurationenactments
```

```shell
kubectl get nnce
```

```
NAME                     AGE
node01.bond0-eth1-eth2   93s
node02.bond0-eth1-eth2   94s
```

```shell
kubectl get nnce node01.bond0-eth1-eth2 -o yaml
```

```yaml
# Output truncated
status:
  conditions:
  - lastHearbeatTime: "2020-01-31T16:15:57Z"
    lastTransitionTime: "2020-01-31T16:15:57Z"
    reason: SuccessfullyConfigured
    status: "False"
    type: Failing
  - lastHearbeatTime: "2020-01-31T16:15:57Z"
    lastTransitionTime: "2020-01-31T16:15:57Z"
    message: successfully reconciled
    reason: SuccessfullyConfigured
    status: "True"
    type: Available
  - lastHearbeatTime: "2020-01-31T16:15:57Z"
    lastTransitionTime: "2020-01-31T16:15:57Z"
    reason: SuccessfullyConfigured
    status: "False"
    type: Progressing
  - lastHearbeatTime: "2020-01-31T16:15:51Z"
    lastTransitionTime: "2020-01-31T16:15:51Z"
    message: All policy selectors are matching the node
    reason: AllSelectorsMatching
    status: "True"
    type: Matching
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

## Removing interfaces

```yaml
removal of the config
```

```shell
execution
```

TODO: alternative with edit

```shell
edit alternative
```

## Restore original configuration

TODO: explain that it is needed to revive eth0, ownership

## Troubleshooting issues

TODO: show failing policy
TODO: show that it reports failing enactment
TODO: show the enactment with the error message
TODO: fix the policy

