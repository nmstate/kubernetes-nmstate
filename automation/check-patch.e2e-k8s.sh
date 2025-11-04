#!/bin/bash -xe

# This script should be able to execute functional tests against Kubernetes
# cluster on any environment with basic dependencies listed in
# check-patch.packages installed and podman / docker running.
#
# yum -y install automation/check-patch.packages
# automation/check-patch.e2e-k8s.sh

teardown() {
    ./cluster/kubectl.sh get nmstate -A -o yaml > $ARTIFACTS/nmstate.yaml || true
    ./cluster/kubectl.sh get pod -n nmstate -o wide > $ARTIFACTS/kubernetes-nmstate.pod.list.txt || true
    for pod in $(./cluster/kubectl.sh get pod -n nmstate -o name); do
        pod_name=$(echo $pod|sed "s#pod/##")
        ./cluster/kubectl.sh -n nmstate logs --prefix=true $pod  > $ARTIFACTS/$pod_name.log || true
        ./cluster/kubectl.sh -n nmstate logs -p --prefix=true $pod  > $ARTIFACTS/$pod_name.previous.log || true
        ./cluster/kubectl.sh -n nmstate describe $pod  > $ARTIFACTS/$pod_name.describe.log || true
    done
    ./cluster/kubectl.sh get events -n nmstate > $ARTIFACTS/nmstate-events.logs || true
    ./cluster/kubectl.sh get events > $ARTIFACTS/cluster-events.logs || true
    make cluster-down
    # Don't fail if there is no logs
    cp -r ${E2E_LOGS}/handler/* ${ARTIFACTS} || true
}

main() {
    export KUBEVIRT_NUM_NODES=5 # 1 control-plane, 4 workers
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

    make E2E_TEST_TIMEOUT=2h E2E_TEST_ARGS="--no-color --output-dir=$ARTIFACTS --junit-report=junit.functest.xml" test-e2e-handler
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
