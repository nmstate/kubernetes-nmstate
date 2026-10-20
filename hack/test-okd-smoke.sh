#!/bin/bash

set -euo pipefail

source ./automation/check-patch.okd.sh

test_dir=$(mktemp -d)
trap 'rm -rf "${test_dir}"' EXIT
export test_dir
export ARTIFACTS="${test_dir}/artifacts"
export KUBECONFIG="${test_dir}/kubeconfig"
export auth_dir="${test_dir}/auth"
mkdir -p "${ARTIFACTS}" "${auth_dir}"
export HELM=test_helm

oc() {
    echo "oc $*" >> "${test_dir}/commands"
    case "$*" in
        "create token"*) echo test-token ;;
        "get nns -o name"*) echo nodenetworkstate/crc ;;
        "wait --for=condition=Available"*) return "${readiness_status:-0}" ;;
        *lastSuccessfulUpdateTime*) return "${report_status:-0}" ;;
    esac
}

test_helm() {
    echo "helm $*" >> "${test_dir}/commands"
}

sleep() { :; }

timeout() {
    echo "timeout $*" >> "${test_dir}/commands"
}

podman() {
    echo "podman $*" >> "${test_dir}/commands"
    if [[ "$1" == login ]]; then
        local token
        read -r token
        test "${token}" = test-token
    fi
    return "${push_status:-0}"
}

export -f oc test_helm sleep timeout podman

# Push only the locally built images, with a temporary scoped credential.
bash -c 'source ./automation/check-patch.okd.sh; push_images'
grep -Fq 'curl --fail --silent --insecure https://127.0.0.1:5000/healthz' "${test_dir}/commands"
grep -q 'create token ci-image-builder -n nmstate --duration=15m' "${test_dir}/commands"
for component in handler operator; do
    grep -q "docker://localhost:5000/nmstate/kubernetes-nmstate-${component}:ci" "${test_dir}/commands"
done
if push_status=1 bash -c 'source ./automation/check-patch.okd.sh; push_images'; then
    echo "Expected a registry failure to fail the lane" >&2
    exit 1
fi

# Exercise the real shared deployment helpers with a mocked cluster.
bash -c 'source ./automation/check-patch.okd.sh; install_operator'
grep -q -- '--set operator.image=image-registry.openshift-image-registry.svc:5000/nmstate/kubernetes-nmstate-operator:ci' "${test_dir}/commands"
grep -q -- '--set handler.image=image-registry.openshift-image-registry.svc:5000/nmstate/kubernetes-nmstate-handler:ci' "${test_dir}/commands"
grep -q -- '--set nmstate.enabled=true' "${test_dir}/commands"
grep -q 'rollout status -w -n nmstate deployment nmstate-operator' "${test_dir}/commands"
grep -q 'rollout status -w -n nmstate ds nmstate-handler' "${test_dir}/commands"
grep -q 'get nns -o name' "${test_dir}/commands"
grep -Fq 'wait --for=jsonpath={.status.numberReady}=1' "${test_dir}/commands"
grep -Fq 'wait --for=jsonpath={.status.lastSuccessfulUpdateTime}' "${test_dir}/commands"

# Failure must propagate; diagnostic collection must not turn a failed lane green.
if readiness_status=1 bash -c 'source ./automation/check-patch.okd.sh; install_operator'; then
    echo "Expected failed NMState readiness to fail installation" >&2
    exit 1
fi
if report_status=1 bash -c 'source ./automation/check-patch.okd.sh; install_operator'; then
    echo "Expected missing node state to fail installation" >&2
    exit 1
fi
(
    trap cleanup EXIT
    exit 42
) && status=0 || status=$?
test "${status}" -eq 42
test ! -e "${test_dir}/auth"

echo "OKD smoke tests passed"
