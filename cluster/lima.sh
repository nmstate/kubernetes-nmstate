#!/bin/bash

LIMA_VM_NAME="${LIMA_VM_NAME:-kubernetes-nmstate}"
LIMA_TEMPLATE="$(dirname "$0")/../lima/kubernetes-nmstate.yaml"

# Detect macOS and re-exec the calling script inside the Lima VM.
# Call this at the top of each cluster script. On Linux it's a no-op.
function lima::ensure_linux() {
    if [[ "$(uname -s)" != "Darwin" ]]; then
        return 0
    fi

    if ! command -v limactl &>/dev/null; then
        echo "ERROR: Lima is required for macOS development." >&2
        echo "Install it with: brew install lima" >&2
        echo "Then create the VM: limactl start ./lima/kubernetes-nmstate.yaml --name ${LIMA_VM_NAME}" >&2
        exit 1
    fi

    if ! limactl list --json 2>/dev/null | grep -q "\"name\":\"${LIMA_VM_NAME}\""; then
        echo "Lima VM '${LIMA_VM_NAME}' not found. Creating it..." >&2
        echo "This will download a VM image and install dependencies (Docker, Go, etc.)." >&2
        echo "First-time setup — this is a one-time operation." >&2
        if ! limactl start "${LIMA_TEMPLATE}" --name "${LIMA_VM_NAME}" --tty=false; then
            echo "ERROR: Lima VM creation failed." >&2
            echo "Serial log (last 50 lines):" >&2
            tail -50 "${HOME}/.lima/${LIMA_VM_NAME}/serial.log" 2>/dev/null || true
            exit 1
        fi
    fi

    local status
    status=$(limactl list --json 2>/dev/null | python3 -c "
import sys, json
for line in sys.stdin:
    obj = json.loads(line)
    if obj.get('name') == '${LIMA_VM_NAME}':
        print(obj.get('status', ''))
        break
" 2>/dev/null || echo "")

    if [[ "$status" != "Running" ]]; then
        echo "Starting Lima VM '${LIMA_VM_NAME}'..." >&2
        limactl start "${LIMA_VM_NAME}"
    fi

    # Unset SSH so Lima doesn't try to use ./cluster/ssh.sh (exported by the
    # Makefile) as its SSH binary — Lima needs the real ssh to connect to the VM.
    unset SSH

    # Re-exec the original script inside the VM, forwarding env vars and args.
    # Use sg to ensure the docker group is active in the session.
    # limactl shell doesn't create a login shell, so supplementary groups
    # from usermod -aG aren't loaded — docker ps would fail with permission denied.
    # Default to amd64 image builds — external clusters are x86_64 and the
    # arm64 Lima VM would otherwise produce arm64 images that can't run there.
    local _archs="${ARCHS:-amd64}"

    exec limactl shell --workdir "$(pwd)" "${LIMA_VM_NAME}" \
        sg docker -c "\
        ARCHS='${_archs}' \
        KUBEVIRT_PROVIDER='${KUBEVIRT_PROVIDER:-}' \
        KUBEVIRT_NUM_NODES='${KUBEVIRT_NUM_NODES:-}' \
        KUBEVIRT_NUM_SECONDARY_NICS='${KUBEVIRT_NUM_SECONDARY_NICS:-}' \
        KUBEVIRT_DEPLOY_PROMETHEUS='${KUBEVIRT_DEPLOY_PROMETHEUS:-}' \
        KUBEVIRT_DEPLOY_GRAFANA='${KUBEVIRT_DEPLOY_GRAFANA:-}' \
        KUBECONFIG='${KUBECONFIG:-}' \
        NM_VERSION='${NM_VERSION:-}' \
        NMSTATE_VERSION='${NMSTATE_VERSION:-}' \
        DEV_IMAGE_REGISTRY='${DEV_IMAGE_REGISTRY:-}' \
        IMAGE_REGISTRY='${IMAGE_REGISTRY:-}' \
        IMAGE_REPO='${IMAGE_REPO:-}' \
        bash $0 $@"
}
