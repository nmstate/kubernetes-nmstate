KUBECONFIG=~/.kube/config

# create a CustomResourceDefinition
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-state-crd.yaml

TEST_NS=test-ns

kubectl create namespace ${TEST_NS}

# start the controller as a daemon
cmd/state-controller/state-controller -kubeconfig ${KUBECONFIG} -n ${TEST_NS}&
sleep 10

# make sure that state CRD is created by the daemon on startup
kubectl --kubeconfig ${KUBECONFIG} get nodenetworkstate ${HOSTNAME} -o yaml -n ${TEST_NS}

# create custom resources based on generated files for different host
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-state-sample.yaml -n ${TEST_NS}

# update the custom state resource based on current hostname
HOSTNAME=`hostname`
sed "s/nodeName: node1/nodeName: ${HOSTNAME}/" manifests/generated/net-state-sample.yaml > tmp.yaml
sed -i "s/name: node1/name: ${HOSTNAME}/" tmp.yaml
kubectl --kubeconfig ${KUBECONFIG} apply -f tmp.yaml -n ${TEST_NS}
rm -f tmp.yaml
sleep 10

# make sure that state is updated in the CRD
kubectl --kubeconfig ${KUBECONFIG} get nodenetworkstate ${HOSTNAME} -o yaml -n ${TEST_NS}

# kill the controller daemon 
kill `ps -o pid,cmd:80 | grep state-controller | grep -v grep | cut -f 1 -d ' '`

# cleanup
kubectl delete namespace ${TEST_NS}
sleep 10