# k8s-node-net-conf
System to configure host networking on Kubernetes using [nmstate](https://nmstate.github.io/).
The system defines 2 CRDs (Custom Resource Definition): NodeNetworkState and NodeNetConfPolicy.
In the project we provide a client that could be invoked manually by an external system (e.g. [Machine Config Operator](https://github.com/openshift/machine-config-operator)) and 2 daemons that can manage the configuration automatically based on the above CRDs.

## Client
The client can run in two modes, as a NodeNetworkState CRD client and a NodeNetConfPolicy CRD client.
### NodeNetworkState Client
This client must have access to nmstate that can operate on the node.

Upon invocation, the client read the list of NodeNetworkState CRD, if no CRD exist for the host it is executed on, it will create one, and fill it with the output of ```nmstatectl show``` as the current status in the CRD.
If a NodeNetworkState CRD exists for the host, it will try to enforce the desired state from the CRD (using: ```nmstatectl set```), and then report back current state.

Note that the desire state could be created based on NodeNetConfPolicy CRDs, or just manually set by an external system.
### NodeNetConfPolicy Client
This client can be executed anywhere in the cluster, and has no dependencies on the host.

The client can run in distributed or centralized mode. In case of distributed (default mode), it will only handle the NodeNetworkState CRD of host it is executed on. In case of centralized mode, there has to be only one location in which the client is executed.
#### Distributed Mode
Upon invocation, the client read the list of NodeNetworkState CRD, as well as the list of NodeNetConfPolicy CRDs.The client will find which NodeNetworkState CRD is for the host it is running on. It will also find all NodeNetConfPolicy CRD that apply to that host (based on their affinity and toleration). Based on the interface match logic they have, and the list of interfaces taken from the hosts NodeNetworkState CRD, it will create aggregated desired state object, and update it into the relevant NodeNetworkState CRD.
#### Centralized mode
___TODO___
## State Controller
This is a privileged host daemon. When it starts it reads the list of NodeNetworkState CRD, if no CRD exist for the host it is executed on, it will create one, and fill it with the output of ```nmstatectl show``` as the current status in the CRD. If a NodeNetworkState CRD exists for the host, it will try to enforce the desired state from the CRD (using: ```nmstatectl set```), and then report back current state.

Whenever it detects an update to the NodeNetworkState CRD which apply to it, it will try to reenforce the current state, and report back the existing one. In case it detects deletion of the NodeNetworkState CRD which apply to it, it will re-create it with current state only.

In case that the enforcement partially or completely failed, the daemon will retry it (with exponential back-off) until it succeeded, and the desired state in the NodeNetworkState CRD is modified.

Even if enforcement was successful, the daemon will periodically poll the current state of the host, and will report it if any modification happened. If such modification is causing the desired state to be different than the current one, it will try to reenforce it (as described above).

If NodeNetworkState CRD is indicated to eb "un-managed" all enforcement and reporting stops.

Note that the desire state could be created based on NodeNetConfPolicy CRDs, or just manually set by an external system.
## Policy Controller
#### Distributed Mode
___TODO___
#### Centralized mode
___TODO___
