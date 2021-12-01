# Prepare environment for kubernetes-nmstate end to end testing. This includes temporary Go paths and binaries.
#
# source automation/check-patch.e2e.setup.sh
# cd ${TMP_PROJECT_PATH}

tmp_dir=/tmp/knmstate/

rm -rf $tmp_dir
mkdir -p $tmp_dir

automation/install-go.sh $tmp_dir

export ARCHS="amd64 arm64"
export TMP_PROJECT_PATH=$tmp_dir/kubernetes-nmstate
export E2E_LOGS=${TMP_PROJECT_PATH}/test_logs/e2e
export ARTIFACTS=${ARTIFACTS-$tmp_dir/artifacts}
export PATH=$tmp_dir/go/bin/:$PATH
mkdir -p $ARTIFACTS


rsync -rt --links $(pwd)/ $TMP_PROJECT_PATH
