#!/bin/bash -xe

previous_minor_version=$(./hack/previous-minor-version.sh)

knmstate_artifact_url="https://github.com/nmstate/kubernetes-nmstate/releases/download/${previous_minor_version}"

test_e2e_updrade_manifests_dir="test/e2e/upgrade/manifests"
test_e2e_updrade_examples_dir="test/e2e/upgrade/examples"

e2e_test_args=$1
E2E_TEST_SUITE_ARGS=$2

mkdir -p $test_e2e_updrade_manifests_dir
mkdir -p $test_e2e_updrade_examples_dir
mkdir -p test_logs/e2e/upgrade

# download example manifests
(
    examples_tar="examples.tar.gz"
    cd $test_e2e_updrade_examples_dir
    curl -k -L "${knmstate_artifact_url}/${examples_tar}" -o $examples_tar
    tar -xvf $examples_tar
    mv ./docs/examples/* .
)

# download manifests for deployment
(
    cd $test_e2e_updrade_manifests_dir
    for manifest in "namespace.yaml" "service_account.yaml" "operator.yaml" "role.yaml" "role_binding.yaml"
    do
        curl -k -L "${knmstate_artifact_url}/$manifest" -o $manifest
    done
)

KUBECONFIG=${KUBECONFIG} OPERATOR_NAMESPACE=${OPERATOR_NAMESPACE} ${GINKGO} $e2e_test_args  $E2E_TEST_SUITE_ARGS ./test/e2e/upgrade
