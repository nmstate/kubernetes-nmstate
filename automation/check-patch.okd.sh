#!/bin/bash

set -euo pipefail

collect_artifacts() {
    oc get nodes -o wide --request-timeout=30s > "${ARTIFACTS}/nodes.txt" 2>&1 || true
    oc get clusteroperators --request-timeout=30s > "${ARTIFACTS}/clusteroperators.txt" 2>&1 || true
    oc get nmstate,nns -o yaml --request-timeout=30s > "${ARTIFACTS}/nmstate.yaml" 2>&1 || true
    oc get events -A --request-timeout=30s > "${ARTIFACTS}/events.txt" 2>&1 || true
    oc get pods -n nmstate -o wide --request-timeout=30s > "${ARTIFACTS}/pods.txt" 2>&1 || true
    local pod
    for pod in $(oc get pods -n nmstate -o name --request-timeout=30s); do
        oc logs -n nmstate "${pod}" --all-containers --request-timeout=30s > "${ARTIFACTS}/${pod#pod/}.log" 2>&1 || true
        oc describe -n nmstate "${pod}" --request-timeout=30s > "${ARTIFACTS}/${pod#pod/}.txt" 2>&1 || true
    done
}

cleanup() {
    local status=$?
    trap - EXIT
    collect_artifacts
    if [[ -n "${registry_pid:-}" ]]; then
        kill "${registry_pid}" 2>/dev/null || true
        wait "${registry_pid}" 2>/dev/null || true
    fi
    rm -rf "${auth_dir}"
    exit "${status}"
}

push_images() {
    oc new-project nmstate
    oc create serviceaccount ci-image-builder -n nmstate
    oc policy add-role-to-user system:image-builder -z ci-image-builder -n nmstate
    oc rollout status -n openshift-image-registry deployment/image-registry --timeout=5m
    oc port-forward -n openshift-image-registry service/image-registry 5000:5000 \
        --address=127.0.0.1 > "${ARTIFACTS}/registry.log" 2>&1 &
    registry_pid=$!
    timeout 60 bash -c 'until curl --fail --silent http://127.0.0.1:5000/healthz; do sleep 1; done'

    # Keep the short-lived, namespace-scoped registry credential out of logs/artifacts.
    # Plain HTTP is confined to loopback; oc tunnels it through the Kubernetes API.
    oc create token ci-image-builder -n nmstate --duration=15m |
        podman login --authfile "${auth_dir}/auth.json" --tls-verify=false \
            --username serviceaccount --password-stdin localhost:5000
    local component
    for component in handler operator; do
        podman manifest push --all --authfile "${auth_dir}/auth.json" --tls-verify=false \
            "localhost:5000/nmstate/kubernetes-nmstate-${component}:ci" \
            "docker://localhost:5000/nmstate/kubernetes-nmstate-${component}:ci"
    done
}

install_operator() {
    export KUBEVIRT_PROVIDER=external
    export KUBEVIRT_NUM_NODES=1
    export DEV_IMAGE_REGISTRY=image-registry.openshift-image-registry.svc:5000
    export OPERATOR_IMAGE="${DEV_IMAGE_REGISTRY}/nmstate/kubernetes-nmstate-operator:ci"
    export HANDLER_IMAGE="${DEV_IMAGE_REGISTRY}/nmstate/kubernetes-nmstate-handler:ci"
    export OPERATOR_NAMESPACE=nmstate HANDLER_NAMESPACE=nmstate
    source ./cluster/sync-operator.sh
    export kubectl=oc

    # Reuse the Helm installation/readiness helpers, without kubevirtci or rebuilding.
    deploy_operator true
    wait_ready_operator true
    oc wait --for=condition=Available nmstate/nmstate --timeout=5m
}

main() {
    : "${KUBECONFIG:?Set KUBECONFIG to the CRC admin kubeconfig}"
    export ARTIFACTS=${ARTIFACTS:-test_logs/okd}
    mkdir -p "${ARTIFACTS}"
    auth_dir=$(mktemp -d)
    trap cleanup EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
    oc wait --for=condition=Ready nodes --all --timeout=10m
    push_images
    install_operator
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
