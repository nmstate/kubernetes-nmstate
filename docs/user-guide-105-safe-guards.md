# 105: Safe Guards

TODO:
- while breaking network connectivity is easy, recovering the node not so much
- knmstate has several mechanisms in place to protect the user from killing cluster nodes
- rollback - when unable to apply config
- apply wrong config, show that it is marked as degraded and host has the original config
- rollout - configuration is applied on one node at the time
- this makes sure we don't kill all the nodes at the same time when configuring the default network, explain how to disable it
- connectivity check - in case the connectivity to the default gateway is lost while configuring, we rollback
- this protects users from killing connectivity which would mean they need to connect directly to the host and manually fix networking

