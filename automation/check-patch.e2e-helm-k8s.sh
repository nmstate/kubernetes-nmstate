#!/bin/bash -xe

# This script validates the Helm chart deployment method end to end:
# helm install (with the chart-created NMState CR), two-step uninstall,
# and then the standard operator e2e suite against manifests rendered
# from the same chart.

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

    # Let's fail fast if generated files differ or the chart does not lint
    make check-gen
    make lint-helm

    # Let's fail fast if it's not compiling
    make operator

    make cluster-down
    make cluster-up
    trap teardown EXIT SIGINT SIGTERM SIGSTOP

    kubectl=./cluster/kubectl.sh

    # Install via the Helm chart; the chart-created NMState CR must bring
    # up the handler (verified inside the sync script).
    make cluster-sync-operator-helm

    # Two-step uninstall (handled inside helm-uninstall): the NMState CR has
    # a finalizer processed by the operator, so it is removed before the
    # chart release.
    make helm-uninstall
    ! $kubectl get deployment -n nmstate nmstate-operator
    ! $kubectl get ds -n nmstate nmstate-handler

    # Hand the cluster to the standard operator e2e suite, which manages
    # its own operator lifecycle from the chart-rendered manifests.
    make cluster-sync-operator
    make E2E_TEST_TIMEOUT=1h E2E_TEST_ARGS="--no-color --output-dir=$ARTIFACTS --junit-report=junit.functest.xml" test-e2e-operator
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
