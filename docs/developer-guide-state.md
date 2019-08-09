# NodeNetworkState controller

Its responsability is to fill-in and update NodeNetworkState currentStatus and
apply desiredStatus if present.

If a `NodeNetworkState` creation event is received it will fill-in currentState
from the node the pod is running on (using : `nmstatectl show`)

In case of `NodeNetworkState` update event with desiredState it will
apply directly the new config into the node (using : `nmstatectl set`)

Although `NodeNetworkState` will apply desired state on the host, it should
never be done by a user. In order to change node network, `NodeNetworkConfigurationPolicy`
must be used.
