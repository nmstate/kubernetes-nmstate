# State Handler
## Deployment
### Manual Installation
In all cases the CRDs has to first be created:
```
# kubectl --kubeconfig ~/.kube/config create -f manifests/generated/net-conf-crd.yaml
# kubectl --kubeconfig ~/.kube/config create -f manifests/generated/net-state-crd.yaml
```
- Client: nmstate has to be installed on the host. To run the client use:
```
# cmd/state-handler -kubeconfig ~/.kube/config -n test-ns -type client -host my-host

```
- Dockerized Client: no need to install nmstate, but docker is needed. To run the client use:
```
docker run -v ~/.kube/:/.kube/ -v /run/dbus/system_bus_socket:/run/dbus/system_bus_socket --rm \
    nmstate/k8s-state-handler -kubeconfig /.kube/config -n test-ns -type client -host my-host
```
- Client Pod: only ```kubectl``` is needed, as the client pod will be scheduled on one of the nodes:
```
kubectl --kubeconfig ~/.kube/config create -f manifests/state-handler-pod.yaml
```
- Controller __TODO__
- Dockerized Controller __TODO__
- Controller pod __TODO__
## Configuration
This is done by modifying the ```desiredState``` object inside the ```NodeNetworkState``` CRD. For example, assuming that the following file (```node1-state.yaml```) has the correct node name (```node1```), and that dummy0 interface exists (and is up) on that node:
```yaml
apiVersion: nmstate.io/v1
kind: NodeNetworkState
metadata:
  creationTimestamp: null
  name: node1
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
```
Calling:
```
kubectl --kubeconfig ~/.kube/config apply -f node1-state.yaml
```
Will set its MTU to 1450. A consequent call to:
```
kubectl --kubeconfig ~/.kube/config get nodenetworkstate node1 -o yaml
```
Should provide with the above ```desiredState``` and well as ```currentState``` that will have (among other interfaces) the ```dummy0``` interface with the new MTS.
### DaemonSet Deployment
The deployment yaml (__TODO__) will create the CRDs, the RBAC, and the necessary daemon sets
- Client: the daemon set only copies and binary into a location from which it can be invoked
- Controller: __TODO__
# Policy Handler
__TODO__