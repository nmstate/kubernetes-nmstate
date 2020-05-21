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
focus='test_id:3796|test_id:3795|test_id:3813|test_id:3794|test_id:3793'
make test/e2e E2E_TEST_TIMEOUT=${TIMEOUT} E2E_TEST_ARGS="--junit-output=junit.functest.xml -ginkgo.noColor -ginkgo.focus $focus" NAMESPACE=$NAMESPACE KUBECONFIG=$KUBECONFIG
