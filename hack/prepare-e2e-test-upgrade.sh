#!/bin/bash -xe

previous_minor_version=$(./hack/previous-minor-version.sh)

knmstate_artifact_url="https://github.com/nmstate/kubernetes-nmstate/releases/download/${previous_minor_version}"

test_e2e_updrade_manifests_dir="test/e2e/upgrade/manifests"
test_e2e_updrade_examples_dir="test/e2e/upgrade/examples"

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

# v0.87.0 published bond.yaml contains "copy-mac-from: eth1", which that
# release's nmstate rejects with InvalidArgument when eth1 is simultaneously
# being enslaved into bond0 during policy application.  Strip only that field
# from the upgrade fixture so that later releases retain normal copy-mac-from
# upgrade coverage.
if [ "${previous_minor_version}" = "v0.87.0" ]; then
    bond_yaml="${test_e2e_updrade_examples_dir}/bond.yaml"
    if ! grep -q 'copy-mac-from:' "${bond_yaml}"; then
        echo "ERROR: expected 'copy-mac-from:' in ${bond_yaml} for v0.87.0 compatibility rewrite, but it was not found" >&2
        exit 1
    fi
    sed -i '/^\s*copy-mac-from:/d' "${bond_yaml}"
fi

# download manifests for deployment
(
    cd $test_e2e_updrade_manifests_dir
    for manifest in "namespace.yaml" "service_account.yaml" "operator.yaml" "role.yaml" "role_binding.yaml"
    do
        curl -k -L "${knmstate_artifact_url}/$manifest" -o $manifest
    done
)
