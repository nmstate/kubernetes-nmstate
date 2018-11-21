KUBECONFIG=~/.kube/config

# create a CustomResourceDefinition
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-state-crd.yaml

TEST_NS=test-ns

echo "=====TEST START: state exists whem client start"
kubectl create namespace ${TEST_NS}

# create custom resources based on generated files for different host
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-state-sample.yaml -n ${TEST_NS}

# create a custom state resource based on current hostname
HOSTNAME=`hostname`
sed "s/nodeName: node1/nodeName: ${HOSTNAME}/" manifests/tests/test-ok.yaml > tmp.yaml
sed -i "s/name: node1/name: ${HOSTNAME}/" tmp.yaml
kubectl --kubeconfig ${KUBECONFIG} create -f tmp.yaml -n ${TEST_NS}
rm -f tmp.yaml

# run the client
cmd/client/client -kubeconfig ${KUBECONFIG} -n ${TEST_NS} -type state -n ${TEST_NS}

# make sure that state is updated in the CRD
echo "=====State should be updated by client"
kubectl --kubeconfig ${KUBECONFIG} get nodenetworkstate ${HOSTNAME} -o yaml -n ${TEST_NS}

# cleanup
kubectl delete namespace ${TEST_NS}
echo "=====TEST END"

sleep 10

echo "=====TEST START: state does not exists when client start"
kubectl create namespace ${TEST_NS}

# create custom resources based on generated files for different host
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-state-sample.yaml -n ${TEST_NS}

# run the client
cmd/client/client -kubeconfig ${KUBECONFIG} -n ${TEST_NS} -type state -n ${TEST_NS}

# make sure that state CRD is created by client
echo "=====State should be created by client"
kubectl --kubeconfig ${KUBECONFIG} get nodenetworkstate ${HOSTNAME} -o yaml -n ${TEST_NS}

# cleanup
kubectl delete namespace ${TEST_NS}
echo "=====TEST END"

sleep 10

echo "=====TEST START: state does not exists when client start in docker"
kubectl create namespace ${TEST_NS}

# create custom resources based on generated files for different host
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-state-sample.yaml -n ${TEST_NS}

# run the client
docker run --net host -v ~/.kube/:/.kube/ -v /run/dbus/system_bus_socket:/run/dbus/system_bus_socket --rm \
    yuvalif/k8s-node-net-conf-client -n ${TEST_NS} -type state

# make sure that state CRD is created by client
echo "=====State should be created by client"
kubectl --kubeconfig ${KUBECONFIG} get nodenetworkstate ${HOSTNAME} -o yaml -n ${TEST_NS}

# cleanup
kubectl delete namespace ${TEST_NS}
echo "=====TEST END"

sleep 10