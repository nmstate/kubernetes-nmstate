# Tutorial: Reporting State

In this example, we will use kubernetes-nmstate to report state of network
interfaces on our cluster nodes. This example will describe two methods to
do that, using active daemon set and on-demand pod client. Read more about
the difference between the two on
[Active vs. On-Demand](user-guide-active-vs-on-demand.md).

## Requirements

Before we start, make sure that you have your Kubernetes/OpenShift cluster
ready with OVS. In order to do that, you can follow guides of deployment on
[local cluster](deployment-local-cluster.md) or your
[arbitrary cluster](deployment-arbitrary-cluster.md).

## Periodically report state from all nodes

Install kubernetes-nmstate state handler as a daemon set (if not done yet). This
daemon set will periodically update reported state of node interfaces. It will
also apply desired specification of node networking if it is changed.

```shell
# on local cluster
./cluster/kubectl.sh create -f _out/manifests/state-controller-ds.yaml

# on arbitrary cluster
kubectl apply -f https://raw.githubusercontent.com/nmstate/kubernetes-nmstate/master/manifests/examples/state-controller-ds.yaml
```

Read Node Network States from all nodes.

```shell
# on local cluster
./cluster/kubectl.sh -n nmstate-default get nodenetworkstates -o yaml

# on arbitrary cluster
kubectl -n nmstate-default get nodenetworkstates -o yaml
```

## Request one-shot report of a state from a single node

Start kubernetes-nmstate state handler in a client mode as a pod on a single
node. This mode might be used kubernetes-nmstate is used as a driver for another
tool configuring nodes.

First, start state handler pod on selected node. (Following examples reflect
`manifests/examples/state-client-pod.yaml` with added `nodeSelector`.)

```yaml
# on local cluster
cat <<EOF | ./cluster/kubectl.sh -n nmstate-default create -f -
apiVersion: v1
kind: Pod
metadata:
 name: state-client
 namespace: nmstate-default
spec:
  nodeSelector:
    hostname: node01
  serviceAccountName: nmstate-state-controller
  containers:
  - name: state-client
    image: registry:5000/kubernetes-nmstate-state-handler:latest
    imagePullPolicy: Always
    args: ["-execution-type", "client"]
    volumeMounts:
    - name: dbus-socket
      mountPath: /run/dbus/system_bus_socket
    env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
    - name: POD_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
    securityContext:
      privileged: true
  volumes:
  - name: dbus-socket
    hostPath:
      path: /run/dbus/system_bus_socket
      type: Socket
EOF

# on arbitrary cluster
cat <<EOF | kubectl -n nmstate-default create -f -
apiVersion: v1
kind: Pod
metadata:
 name: state-client
 namespace: nmstate-default
spec:
  nodeSelector:
    hostname: node01
  serviceAccountName: nmstate-state-controller
  containers:
  - name: state-client
    image: quay.io/nmstate/kubernetes-nmstate-state-handler:latest
    imagePullPolicy: Always
    args: ["-execution-type", "client"]
    volumeMounts:
    - name: dbus-socket
      mountPath: /run/dbus/system_bus_socket
    env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
    - name: POD_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
    securityContext:
      privileged: true
  volumes:
  - name: dbus-socket
    hostPath:
      path: /run/dbus/system_bus_socket
      type: Socket
EOF
```

Then, read reported network state from selected node.

```shell
# on local cluster
./cluster/kubectl.sh -n nmstate-default get nodenetworkstate node01 -o yaml

# on arbitrary cluster
kubectl -n nmstate-default get nodenetworkstate node01 -o yaml
```
