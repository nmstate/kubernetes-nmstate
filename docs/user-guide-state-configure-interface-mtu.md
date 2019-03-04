# Tutorial: Create a Dummy Interfaces and Set its MTU

Use Node Network State to configure a new dummy interfaces on node `node01` and
set its MTU to 1450.

## Requirements

Before we start, make sure that you have your Kubernetes/OpenShift cluster
ready with OVS. In order to do that, you can follow guides of deployment on
[local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

## Configure interface

Install kubernetes-nmstate state handler in active mode (if not already done).

```shell
# on local cluster
./cluster/kubectl.sh create -f _out/manifests/state-controller-ds.yaml

# on arbitrary cluster
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/state-controller-ds.yaml
```

Create `NodeNetworkState` object with desired configuration.

```yaml
# on local cluster
cat <<EOF | ./cluster/kubectl.sh -n nmstate-default create -f -
apiVersion: nmstate.io/v1
kind: NodeNetworkState
metadata:
  name: node01
spec:
  desiredState:
    interfaces:
    - name: dummy0
      state: up
      type: dummy
      mtu: 1450
  managed: true
  nodeName: node1
status:
  currentState:
    capabilities: null
    interfaces: null
EOF

# on arbitrary cluster
cat <<EOF | kubectl -n nmstate-default create -f -
apiVersion: nmstate.io/v1
kind: NodeNetworkState
metadata:
  name: node01
spec:
  desiredState:
    interfaces:
    - name: dummy0
      state: up
      type: dummy
      mtu: 1450
  managed: true
  nodeName: node1
status:
  currentState:
    capabilities: null
    interfaces: null
EOF
```
