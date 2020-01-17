#!/usr/bin/bash -x
#
# Collect the kubernetes-nmstate information from running cluster.

tmp_dir=$(mktemp -d)

namespace=$(kubectl get pods --all-namespaces -l app=kubernetes-nmstate -o jsonpath='{.items[0].metadata.namespace}')
kubectl -n $namespace get pods -l app=kubernetes-nmstate -o jsonpath='{.items[0].spec.containers[0].image}' > $tmp_dir/version
kubectl get nodes -o wide > $tmp_dir/nodes
kubectl -n $namespace get pods -l app=kubernetes-nmstate -o wide | grep nmstate > $tmp_dir/handlers
kubectl get nncp -o yaml > $tmp_dir/nncps.yaml
kubectl get nnce -o yaml > $tmp_dir/nnces.yaml
kubectl get nns -o yaml > $tmp_dir/nnss.yaml
for pod in $(kubectl -n $namespace get pods -l app=kubernetes-nmstate -o jsonpath='{.items[*].metadata.name}'); do
    kubectl -n $namespace logs $pod > $tmp_dir/$pod.log
done

echo "All the information was gathered into $tmp_dir"
