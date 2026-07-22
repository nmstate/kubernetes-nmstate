#!/bin/bash -xe

# This script should be able to execute functional tests against Kubernetes
# cluster on any environment with basic dependencies listed in
# check-patch.packages installed and podman / docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-k8s.sh

teardown() {
    make cluster-down
    # Don't fail if there is no logs
    cp -r ${E2E_LOGS}/operator/* ${ARTIFACTS} || true
}

main() {
    export KUBEVIRT_DEPLOY_PROMETHEUS=false
    export KUBEVIRT_DEPLOY_GRAFANA=false
    export KUBEVIRT_NUM_NODES=3 # 1 control-plane, 2 workers
    source automation/check-patch.setup.sh
    cd ${TMP_PROJECT_PATH}

    operator_namespace=${OPERATOR_NAMESPACE:-nmstate}
    handler_namespace=${HANDLER_NAMESPACE:-nmstate}
    handler_prefix=${HANDLER_PREFIX:-}

    # Let's fail fast if generated files differ or the chart does not lint
    make check-gen
    make lint-helm

    # Let's fail fast if it's not compiling
    make operator

    make cluster-down
    make cluster-up
    trap teardown EXIT SIGINT SIGTERM SIGSTOP

    kubectl=./cluster/kubectl.sh

    # Validate the default local deployment flow end to end: helm install
    # (the chart-created NMState CR must bring up the handler, verified
    # inside the sync script) followed by cluster-clean.
    make cluster-sync
    make cluster-clean
    if $kubectl get deployment -n "${operator_namespace}" nmstate-operator >/dev/null 2>&1; then
        echo "nmstate-operator deployment still exists after cluster-clean"
        exit 1
    fi
    if $kubectl get ds -n "${handler_namespace}" "${handler_prefix}nmstate-handler" >/dev/null 2>&1; then
        echo "nmstate-handler daemonset still exists after cluster-clean"
        exit 1
    fi

    # Hand the cluster to the standard operator e2e suite with just the
    # operator installed; the suite manages NMState lifecycle itself.
    make cluster-sync-operator
    make E2E_TEST_TIMEOUT=1h E2E_TEST_ARGS="--no-color --output-dir=$ARTIFACTS --junit-report=junit.functest.xml" test-e2e-operator
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
