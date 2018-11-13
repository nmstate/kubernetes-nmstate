KUBECONFIG=~/.kube/config

# create a CustomResourceDefinition
kubectl --kubeconfig ${KUBECONFIG} create -f artifacts/examples/crd.yaml

# start the controller as a daemon
cmd/controller/controller -kubeconfig ${KUBECONFIG} &
sleep 1

# create a custom resource of type Foo
kubectl --kubeconfig ${KUBECONFIG} create -f artifacts/examples/example-foo.yaml

# check deployments created through the custom resource
kubectl --kubeconfig ${KUBECONFIG} get deployments

# kill the controller daemon 
kill `ps | grep controller | cut -f 1 -d ' '`

