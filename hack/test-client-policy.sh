KUBECONFIG=~/.kube/config

# create a CustomResourceDefinition
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/examples/configuration-policy-crd.yaml

TEST_NS=test-ns

echo "=====TEST START: policy client - does nothing :-("
kubectl create namespace ${TEST_NS}

# create custom resources based on generated files
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/examples/configuration-policy-example.yaml -n ${TEST_NS}

# run the client
cmd/client/client -kubeconfig ${KUBECONFIG} -n ${TEST_NS} -type policy

# cleanup
kubectl delete namespace ${TEST_NS}
echo "=====TEST END"

sleep 10
