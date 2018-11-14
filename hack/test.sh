KUBECONFIG=~/.kube/config

# create a CustomResourceDefinition
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-state-crd.yaml
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-conf-crd.yaml

# start the controller as a daemon
#cmd/controller/controller -kubeconfig ${KUBECONFIG} &
#sleep 1

TEST_NS=test-ns

kubectl create namespace ${TEST_NS}

# create a custom resources
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-state-sample.yaml -n ${TEST_NS}
kubectl --kubeconfig ${KUBECONFIG} create -f manifests/generated/net-conf-sample.yaml -n ${TEST_NS}

# run the client
cmd/client/client -kubeconfig ${KUBECONFIG} -n ${TEST_NS}

# cleanup
kubectl delete namespace ${TEST_NS}

# kill the controller daemon 
#kill `ps | grep controller | cut -f 1 -d ' '`

