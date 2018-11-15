KUBECONFIG=~/.kube/config

# create a CustomResourceDefinition
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-state-crd.yaml
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-conf-crd.yaml

# start the controller as a daemon
#cmd/controller/controller -kubeconfig ${KUBECONFIG} &
#sleep 1

TEST_NS=test-ns

kubectl create namespace ${TEST_NS}

# create custom resources based on generated files
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-state-sample.yaml -n ${TEST_NS}
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-conf-sample.yaml -n ${TEST_NS}

# create a custom state resource based on current hostname
HOSTNAME=`hostname`
sed "s/nodeName: node1/nodeName: ${HOSTNAME}/" manifests/generated/net-state-sample.yaml > tmp.yaml
sed -i "s/name: node1-network-state/name: tmp-network-state/" tmp.yaml
kubectl --kubeconfig ${KUBECONFIG} create -f tmp.yaml -n ${TEST_NS}
rm -f tmp.yaml

# run the client
cmd/client/client -kubeconfig ${KUBECONFIG} -n ${TEST_NS}

# make sure that state is updated in the CRD
kubectl --kubeconfig ${KUBECONFIG} get nodenetworkstate tmp-network-state -o yaml -n ${TEST_NS}

# cleanup
kubectl delete namespace ${TEST_NS}

# kill the controller daemon 
#kill `ps | grep controller | cut -f 1 -d ' '`

