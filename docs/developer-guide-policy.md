# NodeNetworkConfigurationPolicy controller

This controller runs on every node, when it reconciles `NodeNetworkConfigurationPolicy`,
it checks whether it matches its node. When no `nodeSelector` is specified in the
object, it always matches. Otherwise it compares the selector with node's labels.

The configuration specified in `desiredState` is then saved in matching `NodeNetworkState`.
The `NodeNetworkState` controller is then responsible of configuration.
