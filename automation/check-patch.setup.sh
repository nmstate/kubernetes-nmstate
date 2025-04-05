# Prepare environment for kubernetes-nmstate end to end testing. This includes temporary Go paths and binaries.
#
# source automation/check-patch.e2e.setup.sh
# cd ${TMP_PROJECT_PATH}

tmp_dir=/tmp/knmstate/

. hack/sanitized-xtrace.sh

rm -rf $tmp_dir
mkdir -p $tmp_dir

if gimme --help > /dev/null 2>&1; then

    go_mod_version=$(grep '^go' go.mod |sed 's/go //')
    full_version=$(curl -s https://go.dev/dl/?mode=json |jq -r .[].version |grep "$go_mod_version")
    export GIMME_GO_VERSION=$(echo $full_version |sed 's/go//')
    echo "Installing go $GIMME_GO_VERSION with gimme"
    eval "$(gimme)"
else
    echo "Gimme not installed using existing golang version $(go --version)"
fi

export TMP_PROJECT_PATH=$tmp_dir/kubernetes-nmstate
export E2E_LOGS=${TMP_PROJECT_PATH}/test_logs/e2e
export ARTIFACTS=${ARTIFACTS-$tmp_dir/artifacts}
mkdir -p $ARTIFACTS


rsync -rt --links $(pwd)/ $TMP_PROJECT_PATH
