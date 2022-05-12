./cluster/kubectl.sh delete ippools.v1alpha1.whereabouts.cni.cncf.io -n kube-system --all
./cluster/kubectl.sh delete nncp --all
./cluster/kubectl.sh apply -f test/e2e/whereabouts/static-ip.yaml
