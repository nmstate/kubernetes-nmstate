export KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-'k8s-1.32'}
export KUBEVIRTCI_TAG=2506190727-36f50588
export KUBEVIRT_DEPLOY_PROMETHEUS=${KUBEVIRT_DEPLOY_PROMETHEUS:-true}
export KUBEVIRT_DEPLOY_GRAFANA=${KUBEVIRT_DEPLOY_GRAFANA:-true}

KUBEVIRTCI_REPO='https://github.com/kubevirt/kubevirtci.git'
KUBEVIRTCI_PATH="${PWD}/_kubevirtci"

function kubevirtci::_get_repo() {
    git --git-dir ${KUBEVIRTCI_PATH}/.git remote get-url origin
}

function kubevirtci::_get_tag() {
    git -C ${KUBEVIRTCI_PATH} describe --tags
}

function kubevirtci::install() {
    # Remove cloned kubevirtci repository if it does not match the requested one
    if [[ -d ${KUBEVIRTCI_PATH} ]]; then
        if [[ $(kubevirtci::_get_repo) != ${KUBEVIRTCI_REPO} || $(kubevirtci::_get_tag) != ${KUBEVIRTCI_TAG} ]]; then
            rm -rf ${KUBEVIRTCI_PATH}
        fi
    fi

    if [[ ! -d ${KUBEVIRTCI_PATH} ]]; then
        git clone ${KUBEVIRTCI_REPO} ${KUBEVIRTCI_PATH}
        (
            cd ${KUBEVIRTCI_PATH}
            git checkout ${KUBEVIRTCI_TAG}
        )
    fi
}

function kubevirtci::path() {
    echo -n ${KUBEVIRTCI_PATH}
}

function kubevirtci::kubeconfig() {
    echo -n ${KUBEVIRTCI_PATH}/_ci-configs/${KUBEVIRT_PROVIDER}/.kubeconfig
}
