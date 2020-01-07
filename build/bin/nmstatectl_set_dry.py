#!/usr/bin/env python
#
# nmstatectl set dry run
#
# To check if there is a difference between desired state and actual
# configuration, pass desired state yaml to standard input of this script.
#
# To reduce risks, this should be executed only if the desired configuration is
# the same as in the previous execution.
#
# Although this script may be enough for checking of interfaces, it may not
# always work with routes and DNS.

import sys

import yaml

from libnmstate import metadata
from libnmstate import netinfo
from libnmstate import state


def _get_diff_ifaces(desired_state_raw):
    desired_state = state.State(yaml.safe_load(desired_state_raw))
    current_state = state.State(netinfo.show())

    desired_state.sanitize_ethernet(current_state)
    desired_state.sanitize_dynamic_ip()
    desired_state.merge_routes(current_state)
    desired_state.merge_dns(current_state)
    desired_state.merge_route_rules(current_state)
    desired_state.remove_unknown_interfaces()
    metadata.generate_ifaces_metadata(desired_state, current_state)

    state2edit = state.create_state(
        desired_state.state,
        interfaces_to_filter=(set(current_state.interfaces))
    )
    state2edit.merge_interfaces(current_state)

    return set(state2edit.interfaces)


def _main():
    diff_ifaces = _get_diff_ifaces(sys.stdin.read())

    if diff_ifaces:
        print(f'Some interfaces need to be edited: {diff_ifaces}', file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    _main()
