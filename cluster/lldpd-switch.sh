#!/bin/bash
#
# Emulate a production top-of-rack switch for the kubevirtci cluster.
#
# Production nodes receive LLDPDUs from the switch they are cabled to: the
# frames arrive inbound on the NIC with the switch port as source MAC. The
# kubevirtci cluster network has no LLDP-speaking switch, so nodes never see
# any LLDP neighbor (the bridges also do not forward the link-local scoped
# LLDP group address between nodes, just like a compliant switch).
#
# To reproduce the production behavior without changing kubevirtci, run
# lldpd in an extra container that joins the cluster network namespace
# (owned by the ${KUBEVIRT_PROVIDER}-dnsmasq container) bound to the tap
# devices that connect the node VM NICs (tapXX -> primary NIC, stapX-Y ->
# secondary NICs). lldpd then transmits LLDPDUs on every "switch port" and
# each node receives them inbound on its NICs, exactly like from a real
# switch.

set -ex

source ./cluster/kubevirtci.sh

DNSMASQ_CONTAINER="${KUBEVIRT_PROVIDER}-dnsmasq"
LLDPD_SWITCH_CONTAINER="${KUBEVIRT_PROVIDER}-lldpd-switch"
LLDPD_SWITCH_IMAGE="localhost/kubernetes-nmstate-lldpd-switch:latest"

# Same container runtime detection kubevirtci cluster-up uses, so we talk to
# the runtime that owns the cluster containers.
function detect_cri() {
    local podman_socket=${KUBEVIRTCI_PODMAN_SOCKET:-/run/podman/podman.sock}
    if [ "${KUBEVIRTCI_RUNTIME:-}" = "podman" ]; then
        echo "podman --remote --url=unix://${podman_socket}"
    elif [ "${KUBEVIRTCI_RUNTIME:-}" = "docker" ]; then
        echo "docker"
    elif curl --unix-socket "${podman_socket}" http://d/v3.0.0/libpod/info >/dev/null 2>&1; then
        echo "podman --remote --url=unix://${podman_socket}"
    elif docker ps >/dev/null 2>&1; then
        echo "docker"
    fi
}

function up() {
    if ! ${CRI} inspect "${DNSMASQ_CONTAINER}" >/dev/null 2>&1; then
        echo "no ${DNSMASQ_CONTAINER} container found, skipping LLDP switch emulation"
        return 0
    fi
    down

    # Install lldpd at image build time: containers joining another container
    # network namespace do not get working DNS in every environment (e.g.
    # CI), so the switch container must not do any network I/O at runtime.
    # Build with the host network so the build container resolves names with
    # the host resolver (the CI package proxy hostname is only resolvable
    # there).
    ${CRI} build --network host -t "${LLDPD_SWITCH_IMAGE}" -f cluster/lldpd-switch.Dockerfile cluster/

    ${CRI} run -d --name "${LLDPD_SWITCH_CONTAINER}" \
        --cap-add=NET_ADMIN --cap-add=NET_RAW \
        --network "container:${DNSMASQ_CONTAINER}" \
        "${LLDPD_SWITCH_IMAGE}" \
        sh -c '
            set -ex
            echo "configure lldp tx-interval 5" > /etc/lldpd.conf
            echo "configure system hostname lldp-switch" >> /etc/lldpd.conf
            # Transmit on the bridge ports attached to the node VM NICs,
            # like a real switch does on each of its ports. Exact interface
            # names make lldpd accept the tap devices unconditionally.
            ports=$(ls /sys/class/net | grep -E "^(tap[0-9]+|stap[0-9]+-[0-9]+)$" | tr "\n" "," | sed "s/,$//")
            if [ -z "${ports}" ]; then
                echo "no tap/stap bridge ports found in the cluster network namespace" >&2
                exit 1
            fi
            exec lldpd -d -I "${ports}"
        '

    # lldpd starts in the background; fail cluster-up right away instead of
    # timing out later in the LLDP e2e tests if it did not come up.
    for _ in $(seq 1 30); do
        if [ "$(${CRI} inspect -f '{{.State.Running}}' "${LLDPD_SWITCH_CONTAINER}" 2>/dev/null)" != "true" ]; then
            break
        fi
        if ${CRI} exec "${LLDPD_SWITCH_CONTAINER}" pgrep lldpd >/dev/null 2>&1; then
            return 0
        fi
        sleep 2
    done
    ${CRI} logs "${LLDPD_SWITCH_CONTAINER}" || true
    echo "the LLDP switch emulation container failed to start lldpd" >&2
    exit 1
}

function down() {
    ${CRI} rm -f "${LLDPD_SWITCH_CONTAINER}" >/dev/null 2>&1 || true
}

CRI=$(detect_cri)
if [ -z "${CRI}" ]; then
    echo "no working container runtime found, skipping LLDP switch emulation"
    exit 0
fi

case "${1:-up}" in
up)
    up
    ;;
down)
    down
    ;;
*)
    echo "usage: $0 up|down" >&2
    exit 1
    ;;
esac
