#!/bin/bash -xe

# This script should be able to execute functional tests against Kubernetes
# cluster on any environment with basic dependencies listed in
# check-patch.packages installed and docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-k8s.sh

teardown() {
    ./cluster/kubectl.sh get pod -n nmstate -o wide > $ARTIFACTS/kubernetes-nmstate.pod.list.txt || true
    ./cluster/kubectl.sh logs --tail=1000 -n nmstate -l app=kubernetes-nmstate > $ARTIFACTS/kubernetes-nmstate.pod.logs || true
    make cluster-down
    # Don't fail if there is no logs
    cp ${E2E_LOGS}/handler/*.log ${ARTIFACTS} || true
}

main() {
    export KUBEVIRT_PROVIDER='k8s-1.20'
    export KUBEVIRT_NUM_NODES=3 # 1 master, 2 workers
    source automation/check-patch.setup.sh
    cd ${TMP_PROJECT_PATH}

    # Let's fail fast if generated files differ
    make check-gen

    # Let's fail fast if it's not compiling
    make handler

    make cluster-down
    make cluster-up
    trap teardown EXIT SIGINT SIGTERM SIGSTOP
    make cluster-sync

    export E2E_TEST_SUITE_ARGS="--junit-output=$ARTIFACTS/junit.functest.xml"
    if [ "$NMSTATE_PARALLEL_ROLLOUT" == "true" ]; then
       E2E_TEST_SUITE_ARGS="${E2E_TEST_SUITE_ARGS} -ginkgo.skip='user-guide|nns|sequential'"
    else
       E2E_TEST_SUITE_ARGS="${E2E_TEST_SUITE_ARGS} -ginkgo.skip='parallel'"
    fi

    make E2E_TEST_TIMEOUT=1h E2E_TEST_ARGS="-noColor" test-e2e-handler
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
