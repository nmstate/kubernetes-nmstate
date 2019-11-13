# Prepare environment for kubernetes-nmstate end to end testing. This includes temporary Go paths and binaries.
#
# source automation/check-patch.e2e.setup.sh
# cd ${TMP_PROJECT_PATH}

echo 'Setup Go paths'
export GOROOT=/tmp/kubernetes-nmstate/go/root
mkdir -p $GOROOT
export GOPATH=/tmp/kubernetes-nmstate/go/path
mkdir -p $GOPATH
export PATH=${GOPATH}/bin:${GOROOT}/bin:${PATH}

export GIMME_GO_VERSION=1.12
echo "Install Go $GIMME_GO_VERSION"
GIMME=/tmp/kubernetes-nmstate/go/gimme
mkdir -p $GIMME
curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | HOME=${GIMME} bash >> ${GIMME}/gimme.sh
source ${GIMME}/gimme.sh

echo 'Install operator repository under the temporary Go path'
TMP_PROJECT_PATH=${GOPATH}/src/github.com/nmstate/kubernetes-nmstate
rm -rf ${TMP_PROJECT_PATH}
mkdir -p ${TMP_PROJECT_PATH}
cp -rf $(pwd)/* ${TMP_PROJECT_PATH}

echo 'Exporting temporary project path'
export TMP_PROJECT_PATH
