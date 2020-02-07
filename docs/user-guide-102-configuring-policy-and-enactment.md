# Introduction: Configuration

TODO: what are policies and enactments

TODO: desired state API

## Creating interfaces

<!-- When updating following example, don't forget to update respective attached file -->

[Download example](examples/bond0-eth1-eth2_up.yaml)

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
kubectl apply -f bond0-eth1-eth2_up.yaml
```

```shell
kubectl wait nncp bond0-eth1-eth2 --for condition=Available --timeout 2m
```

```shell
kubectl get nodenetworkconfigurationpolicies
```

```
NAME               STATUS
bond0-eth1-eth2   SuccessfullyConfigured
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
# output truncated
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

TODO About enactments.

You see it is applied to two nodes.

```shell
kubectl get nodenetworkconfigurationenactments
```

```
NAME                     AGE
node01.bond0-eth1-eth2   93s
node02.bond0-eth1-eth2   94s
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
# output truncated
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

TODO you may expect that the configuration would be removed by removing policy,
that's not the case. knmstate does now own any configuration, NM does. If you
want to remove stuff, you gotta explicitly create a policy to do so.

<!-- When updating following example, don't forget to update respective attached file -->

[Download example](examples/bond0-eth1-eth2_absent.yaml)

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: bond0-eth1-eth2
spec:
  desiredState:
    interfaces:
    - name: bond0
      state: absent
```

```shell
kubectl apply -f bond0-eth1-eth2_absent.yaml
```

```shell
kubectl wait nncp bond0-eth1-eth2 --for condition=Available --timeout 2m
```

> **Note:** TODO: alternative with edit, following will open an editor, all you have to do is to edit up to absent.
>
> ```shell
> kubectl edit nncp bond0-eth1-eth2
> ```

```shell
kubectl delete nncp bond0-eth1-eth2
```

## Restore original configuration

TODO There is another behavior that may surprise you. By removing the bonding,
original configuration of its slaves is not restored. Instead they will be
detached from the bonding and left down.

```shell
kubectl get nns node01 -o yaml
```

```yaml
# output truncated
status:
  currentState:
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

There is no history kept, one of the reasons is that we don't want to decide for
the administrator what should happen, it all should be explicit. In order to
setup IP on those NICs again, one can create a new policy:

<!-- When updating following example, don't forget to update respective attached file -->

[Download example](examples/eth1-eth2_up.yaml)

```yaml
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: eth1
spec:
  desiredState:
    interfaces:
    - name: eth1
      type: ethernet
      state: up
      ipv4:
        dhcp: true
        enabled: true
---
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: eth2
spec:
  desiredState:
    interfaces:
    - name: eth2
      type: ethernet
      state: up
      ipv4:
        dhcp: true
        enabled: true
```

```shell
kubectl apply -f eth1-eth2_up.yaml
```

```shell
kubectl wait nncp eth1 eth2 --for condition=Available --timeout 2m
```

The interfaces are back again.

```shell
kubectl get nns node01 -o yaml
```

```yaml
# output truncated
status:
  currentState:
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

## Troubleshooting issues

So far we went only through successful configurations.

TODO: show failing policy

<!-- When updating following example, don't forget to update respective attached file -->

[Download example](examples/eth666_up.yaml)

```
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: eth666
spec:
  desiredState:
    interfaces:
    - name: eth666
      type: ethernet
      state: up
```

```shell
kubectl apply -f eth666_up.yaml
```

```shell
kubectl wait nncp eth666 --for condition=Available --timeout 30s
```

TODO: see that it failed, let's see why

TODO: show that one of the policies is failing

```shell
kubectl get nncp
```

```
NAME     STATUS
eth1     SuccessfullyConfigured
eth2     SuccessfullyConfigured
eth666   FailedToConfigure
```

TODO has `FailedToConfigure`

TODO: show that it reports failing enactment

```shell
kubectl get nnce
```

```
NAME            STATUS
node01.eth1     SuccessfullyConfigured
node01.eth2     SuccessfullyConfigured
node01.eth666   FailedToConfigure
node02.eth1     SuccessfullyConfigured
node02.eth2     SuccessfullyConfigured
node02.eth666   FailedToConfigure
```

On both nodes

TODO: show the enactment with the error message

```shell
kubectl get nnce node01.eth666 -o yaml
```

```yaml
# output truncated
status:
  conditions:
  - lastHearbeatTime: "2020-02-07T09:17:36Z"
    lastTransitionTime: "2020-02-07T09:17:36Z"
    message: |-
      error reconciling NodeNetworkConfigurationPolicy at desired state apply: , failed to execute nmsta
tectl set --no-commit --timeout 240: 'exit status 1' '' '2020-02-07 09:17:30,636 root         DEBUG    C
heckpoint /org/freedesktop/NetworkManager/Checkpoint/7 created for all devices: 240                    
      2020-02-07 09:17:30,637 root         DEBUG    Adding new interfaces: ['eth666']
      2020-02-07 09:17:30,637 root         DEBUG    Connection settings for ConnectionSetting.create:
      id: eth666
      iface: eth666
      uuid: 349af19a-006c-4f95-b06b-5baf3c673abb
      type: 802-3-ethernet
      autoconnect: True
      autoconnect_slaves: <enum NM_SETTING_CONNECTION_AUTOCONNECT_SLAVES_YES of type NM.SettingConnectio
nAutoconnectSlaves>                                                                                    
      2020-02-07 09:17:30,638 root         DEBUG    Editing interfaces: []
      2020-02-07 09:17:30,641 root         DEBUG    Executing NM action: func=add_connection_async
      2020-02-07 09:17:30,648 root         DEBUG    Connection adding succeeded: dev=eth666
      2020-02-07 09:17:30,648 root         DEBUG    Executing NM action: func=safe_activate_async
      2020-02-07 09:17:30,649 root         ERROR    NM main-loop aborted: Connection activation failed o
n connection_id eth666: error=nm-manager-error-quark: No suitable device found for this connection (devi
ce cni0 not available because profile is not compatible with device (mismatching interface name)). (3) 
      2020-02-07 09:17:30,653 root         DEBUG    Checkpoint /org/freedesktop/NetworkManager/Checkpoin
t/7 rollback executed: dbus.Dictionary({dbus.String('/org/freedesktop/NetworkManager/Devices/7'): dbus.U
Int32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/8'): dbus.UInt32(0), dbus.String('/org/fr
eedesktop/NetworkManager/Devices/10'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devi
ces/5'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/4'): dbus.UInt32(0), dbus.
String('/org/freedesktop/NetworkManager/Devices/9'): dbus.UInt32(0), dbus.String('/org/freedesktop/Netwo
rkManager/Devices/6'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/2'): dbus.UI
nt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/1'): dbus.UInt32(0), dbus.String('/org/fre
edesktop/NetworkManager/Devices/3'): dbus.UInt32(0)}, signature=dbus.Signature('su'))                  
      Traceback (most recent call last):
        File "/usr/bin/nmstatectl", line 11, in <module>
          load_entry_point('nmstate==0.2.2', 'console_scripts', 'nmstatectl')()
        File "/usr/lib/python3.7/site-packages/nmstatectl/nmstatectl.py", line 59, in main
          return args.func(args)
        File "/usr/lib/python3.7/site-packages/nmstatectl/nmstatectl.py", line 221, in apply
          return apply_state(statedata, args.verify, args.commit, args.timeout)
        File "/usr/lib/python3.7/site-packages/nmstatectl/nmstatectl.py", line 237, in apply_state
          checkpoint = libnmstate.apply(state, verify_change, commit, timeout)
        File "/usr/lib/python3.7/site-packages/libnmstate/netapplier.py", line 66, in apply
          state.State(desired_state), verify_change, commit, rollback_timeout
        File "/usr/lib/python3.7/site-packages/libnmstate/netapplier.py", line 148, in _apply_ifaces_sta
te                                                                                                     
          con_profiles=ifaces_add_configs + ifaces_edit_configs,
        File "/usr/lib64/python3.7/contextlib.py", line 119, in __exit__
          next(self.gen)
        File "/usr/lib/python3.7/site-packages/libnmstate/netapplier.py", line 215, in _setup_providers
          mainloop.error
      libnmstate.error.NmstateLibnmError: Unexpected failure of libnm when running the mainloop: run exe
cution                                                                                                 
      '
    reason: FailedToConfigure
    status: "True"
    type: Failing
```

TODO: currently we show the full failed configuration output, the interesting line there is:

```
Connection activation failed on connection_id eth666: error=nm-manager-error-quark: No suitable device found for this connection
```

As 

TODO: absent the policy

<!-- When updating following example, don't forget to update respective attached file -->

[Download example](examples/eth666_absent.yaml)

```
apiVersion: nmstate.io/v1alpha1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: eth666
spec:
  desiredState:
    interfaces:
    - name: eth666
      state: absent
```

```shell
kubectl apply -f eth666_absent.yaml
```

```shell
kubectl wait nncp eth666 --for condition=Available --timeout 2m
```

TODO: remove the policy

```shell
kubectl delete nncp eth666
```

<!-- TODO: Link the next introduction article once it is introduced --> 
