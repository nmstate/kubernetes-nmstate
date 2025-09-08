---
title: "Troubleshooting"
tag: "user-guide"
---

Node network configuration is a risky business. A lot can go wrong and when it
does, it can render the whole node unreachable and non-operational. This guide
will show you how to obtain information about failed configuration and how does
the operator protect the user from breaking the cluster networking.

## Invalid configuration

If any of the following cases render the configuration faulty, the setup will be
automatically rolled back and Enactment will report the failure. In addition,
an event for the NodeNetworkConfigurationPolicy will be created.

* Configuration fails to be applied on the host (due to missing interfaces, inability to obtain IP, invalid attributes, ...)
* Connectivity to the default gateway is broken
* Connectivity to the API server is broken

In the following example, we will create a Policy configuring an unavailable
interface and observe the results:

<!-- When updating following example, don't forget to update respective attached file -->

[Download example]({{ "user-guide/eth666_up.yaml" | relative_url }})

```yaml
{% include_absolute 'user-guide/eth666_up.yaml' %}
```

```shell
kubectl apply -f eth666_up.yaml
```

Let's wait for `Degraded` condition since we anticipate the Policy to fail.
Usually, one would wait for `Available` until the timeout.

```shell
kubectl wait nncp eth666 --for condition=Degraded --timeout 2m
```

We can list the Enactments to see why we are in the `Degraded` state:

```shell
kubectl get nnce
```

```
NAME            STATUS
node01.eth666   Failing
node02.eth666   Failing
```

Both Enactments have `FailedToConfigure`, let's see why:

```shell
kubectl get nnce node01.eth666 -o yaml
```

```
# output truncated
status:
  conditions:
  - lastHeartbeatTime: "2020-02-08T20:23:15Z"
    lastTransitionTime: "2020-02-08T20:23:15Z"
    message: |-
      error reconciling NodeNetworkConfigurationPolicy at desired state apply: , failed to execute nmstatectl set --no-commit --timeout 240: 'exit status 1' '' '2020-02-08 20:23:09,563 root         DEBUG    Checkpoint /org/freedesktop/NetworkManager/Checkpoint/3 created for all devices: 240
      2020-02-08 20:23:09,564 root         DEBUG    Adding new interfaces: ['eth666']
      2020-02-08 20:23:09,564 root         DEBUG    Connection settings for ConnectionSetting.create:
      id: eth666
      iface: eth666
      uuid: 70b556e5-e2da-4e46-a98f-531bc73d0bf5
      type: 802-3-ethernet
      autoconnect: True
      autoconnect_slaves: <enum NM_SETTING_CONNECTION_AUTOCONNECT_SLAVES_YES of type NM.SettingConnectionAutoconnectSlaves>
      2020-02-08 20:23:09,565 root         DEBUG    Editing interfaces: []
      2020-02-08 20:23:09,567 root         DEBUG    Executing NM action: func=add_connection_async
      2020-02-08 20:23:09,572 root         DEBUG    Connection adding succeeded: dev=eth666
      2020-02-08 20:23:09,572 root         DEBUG    Executing NM action: func=safe_activate_async
      2020-02-08 20:23:09,573 root         ERROR    NM main-loop aborted: Connection activation failed on connection_id eth666: error=nm-manager-error-quark: No suitable device found for this connection (devicecni0 not available because profile is not compatible with device (mismatching interface name)). (3)      2020-02-08 20:23:09,576 root         DEBUG    Checkpoint /org/freedesktop/NetworkManager/Checkpoint/3 rollback executed: dbus.Dictionary({dbus.String('/org/freedesktop/NetworkManager/Devices/9'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/10'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/5'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/8'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/3'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/7'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/2'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/6'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/1'): dbus.UInt32(0), dbus.String('/org/freedesktop/NetworkManager/Devices/4'): dbus.UInt32(0)}, signature=dbus.Signature('su'))
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
        File "/usr/lib/python3.7/site-packages/libnmstate/netapplier.py", line 148, in _apply_ifaces_state
          con_profiles=ifaces_add_configs + ifaces_edit_configs,
        File "/usr/lib64/python3.7/contextlib.py", line 119, in __exit__
          next(self.gen)
        File "/usr/lib/python3.7/site-packages/libnmstate/netapplier.py", line 215, in _setup_providers
          mainloop.error
      libnmstate.error.NmstateLibnmError: Unexpected failure of libnm when running the mainloop: run execution
      '
    reason: FailedToConfigure
    status: "True"
    type: Failing
```

The message in `Failing` condition is currently a little bloated since it
contains the whole error output of a failed call. The interesting message is in
the `ERROR` log line:

```
Connection activation failed on connection_id eth666: error=nm-manager-error-quark: No suitable device found for this connection
```

In addition, events are generated in the default namespace:
```shell
$ kubectl get events | grep Warning
6m55s       Warning   ReconcileFailed           nodenetworkconfigurationpolicy/eth666   error reconciling NodeNetworkConfigurationPolicy on node node02 at desired state apply: "",...
```

And these events are also linked to the NodeNetworkConfigurationPolicy when using `kubectl describe`:
```shell
kubectl describe nodenetworkconfigurationpolicies.nmstate.io eth666
```

```
# output truncated
Events:
  Type     Reason           Age    From                    Message
  ----     ------           ----   ----                    -------
  Warning  ReconcileFailed  8m25s  node02.nmstate-handler  error reconciling NodeNetworkConfigurationPolicy on node node02 at desired state apply: "",
(...)
NmstateError: InvalidArgument: Ethernet interface eth666 does not exists
'
```

The configuration therefore failed due to absence of NIC `eth666` on the node.
Now we can either fix the Policy to edit an available interface or safely remove
it:

```
kubectl delete nncp eth666
```

## Enabling debug logging

For advanced troubleshooting, you can enable verbose debug logging by configuring the `LogLevel` field in the NMState custom resource. This will provide more detailed output from nmstatectl operations, which can be helpful when diagnosing complex networking issues.

### Setting debug log level

To enable debug logging, update the NMState CR with `logLevel: debug`:

```yaml
apiVersion: nmstate.io/v1
kind: NMState
metadata:
  name: nmstate
spec:
  logLevel: debug  # Valid values: "info" (default) or "debug"
```

Apply the configuration:

```shell
kubectl patch nmstate nmstate --type='merge' -p='{"spec":{"logLevel":"debug"}}'
```

### Verifying debug mode

You can verify that debug mode is active by checking the handler DaemonSet pods:

```shell
kubectl logs -n nmstate -l component=kubernetes-nmstate-handler --tail=50
```

When debug mode is enabled, you should see more verbose output including:
- Detailed nmstatectl command execution logs
- Verbose nmstate operations with `-vv` flag
- Additional debugging information from network configuration operations

### Reverting to info log level

To reduce log verbosity, revert to info log level:

```shell
kubectl patch nmstate nmstate --type='merge' -p='{"spec":{"logLevel":"info"}}'
```

**Note:** Debug logging significantly increases log volume and should primarily be used for troubleshooting purposes. Consider reverting to info level after resolving issues to maintain optimal performance.

## Continue reading

This was the last article from the introduction series. You can continue reading
specific recipes on how to configure various interface types. You will find them
in the [Deployment]({{ "deployment.html" | relative_url }}) or [Examples]({{ "examples.html" | relative_url }}) pages.
