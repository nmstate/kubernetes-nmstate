# Prepare environment for kubernetes-nmstate end to end testing. This includes temporary Go paths and binaries.
#
# source automation/check-patch.e2e.setup.sh
# cd ${TMP_PROJECT_PATH}

tmp_dir=/tmp/knmstate/

rm -rf $tmp_dir

echo 'Setup Go paths'
export GOROOT=$tmp_dir/go/root
mkdir -p $GOROOT
export PATH=${GOROOT}/bin:${PATH}

export GIMME_GO_VERSION=$(grep "^go " go.mod |awk '{print $2}')
echo "Install Go $GIMME_GO_VERSION"
gimme_dir=$tmp_dir/go/gimme
mkdir -p $gimme_dir
curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | HOME=${gimme_dir} bash >> ${gimme_dir}/gimme.sh
source $gimme_dir/gimme.sh

export TMP_PROJECT_PATH=$tmp_dir/kubernetes-nmstate
rsync -rt --links --filter=':- .gitignore' $(pwd)/ $TMP_PROJECT_PATH
