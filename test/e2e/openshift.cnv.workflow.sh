#!/bin/bash

set -xe

# Configure kubeconfig
export KUBECONFIG=${KUBECONFIG:-$HOME/oc4/working/auth/kubeconfig}
export KUBECTL=${KUBECTL:-oc}
export NAMESPACE=${NAMESPACE:-openshift-cnv}
export SSH=${SSH:-./ssh.sh}
export PRIMARY_NIC=${PRIMARY_NIC:-ens3}
export FIRST_SECONDARY_NIC=${FIRST_SECONDARY_NIC:-ens7}
export SECOND_SECONDARY_NIC=${SECOND_SECONDARY_NIC:-ens8}
export TIMEOUT=${TIMEOUT:-60m}

if [ ! -f  $SSH ]; then
    cat << EOF > ${SSH}
#!/bin/bash
node_name=\${1}
node_ip=\$($KUBECTL get node \${node_name} --no-headers -o wide | awk '{print \$6}')
ssh core@\${node_ip} -- \${@:3}
EOF
    chmod +x ${SSH}
fi

# Run workflow tests
FOCUS_1='Nodes.*when.*are.*up.*and.*new.*interface.*is.*configured.*should.*update.*node.*network.*state.*with.*it'
FOCUS_2='rollback.*when.*connectivity.*to.*default.*gw.*is.*lost.*after.*state.*configuration.*should.*rollback.*to.*a.*good.*gw.*configuration'
FOCUS_3='NodeSelector.*when.*policy.*is.*set.*with.*node.*selector.*not.*matching.*any.*nodes.*should.*not.*update.*any.*nodes.*and.*have.*false.*Matching.*state'
FOCUS_4='EnactmentCondition.*when.*applying.*valid.*config.*should.*go.*from.*Progressing.*to.*Available'
make test/e2e E2E_TEST_TIMEOUT=${TIMEOUT} E2E_TEST_ARGS="-ginkgo.focus $FOCUS_1|$FOCUS_2|$FOCUS_3|$FOCUS_4" NAMESPACE=$NAMESPACE KUBECONFIG=$KUBECONFIG
