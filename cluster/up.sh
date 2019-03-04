#!/bin/bash -e

KUBERNETES_IMAGE="k8s-1.11.0@sha256:3412f158ecad53543c9b0aa8468db84dd043f01832a66f0db90327b7dc36a8e8"
OPENSHIFT_IMAGE="os-3.11.0-crio@sha256:3f11a6f437fcdf2d70de4fcc31e0383656f994d0d05f9a83face114ea7254bc0"

KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-k8s-1.11.0}
KUBEVIRT_MEMORY_SIZE=${KUBEVIRT_MEMORY_SIZE:-5120M}
KUBEVIRT_NUM_NODES=${KUBEVIRT_NUM_NODES:-1}

SECONDARY_NICS_NUM=${SECONDARY_NICS_NUM:-1}

if ! [[ $KUBEVIRT_NUM_NODES =~ '^-?[0-9]+$' ]] || [[ $KUBEVIRT_NUM_NODES -lt 1 ]] ; then
    KUBEVIRT_NUM_NODES=1
fi

case "${KUBEVIRT_PROVIDER}" in
    'k8s-1.11.0')
        image=$KUBERNETES_IMAGE
        ;;
    'os-3.11.0')
        image=$OPENSHIFT_IMAGE
        ;;
esac

CREATE_SECONDARY_NICS=""
for i in $(seq 1 ${SECONDARY_NICS_NUM}); do
    CREATE_SECONDARY_NICS+=" -device virtio-net-pci,netdev=secondarynet$i -netdev tap,id=secondarynet$i,ifname=stap$i,script=no,downscript=no"
done

echo "Install cluster from image: ${image}"
if [[ $image == $KUBERNETES_IMAGE ]]; then
    # Run Kubernetes cluster image
    ./cluster/cli.sh run --random-ports --nodes ${KUBEVIRT_NUM_NODES} --memory ${KUBEVIRT_MEMORY_SIZE} --background --qemu-args "'${CREATE_SECONDARY_NICS}'" kubevirtci/${image}

    # Copy kubectl tool and configuration file
    ./cluster/cli.sh scp /usr/bin/kubectl - > ./cluster/.kubectl
    chmod u+x ./cluster/.kubectl
    ./cluster/cli.sh scp /etc/kubernetes/admin.conf - > ./cluster/.kubeconfig

    # Configure insecure access to Kubernetes cluster
    cluster_port=$(./cluster/cli.sh ports k8s | tr -d '\r')
    ./cluster/kubectl.sh config set-cluster kubernetes --server=https://127.0.0.1:$cluster_port
    ./cluster/kubectl.sh config set-cluster kubernetes --insecure-skip-tls-verify=true
elif [[ $image == $OPENSHIFT_IMAGE ]]; then
    # If on a developer setup, expose ocp on 8443, so that the openshift web console can be used (the port is important because of auth redirects)
    if [ -z "${JOB_NAME}" ]; then
        KUBEVIRT_PROVIDER_EXTRA_ARGS="${KUBEVIRT_PROVIDER_EXTRA_ARGS} --ocp-port 8443"
    fi

    # Run OpenShift cluster image
    ./cluster/cli.sh run --random-ports --reverse --nodes ${KUBEVIRT_NUM_NODES} --memory ${KUBEVIRT_MEMORY_SIZE} --background --qemu-args "'${CREATE_SECONDARY_NICS}'" kubevirtci/${image} ${KUBEVIRT_PROVIDER_EXTRA_ARGS}
    ./cluster/cli.sh scp /etc/origin/master/admin.kubeconfig - > ./cluster/.kubeconfig
    ./cluster/cli.sh ssh node01 -- sudo cp /etc/origin/master/admin.kubeconfig ~vagrant/
    ./cluster/cli.sh ssh node01 -- sudo chown vagrant:vagrant ~vagrant/admin.kubeconfig

    # Copy oc tool and configuration file
    ./cluster/cli.sh scp /usr/bin/oc - > ./cluster/.kubectl
    chmod u+x ./cluster/.kubectl
    ./cluster/cli.sh scp /etc/origin/master/admin.kubeconfig - > ./cluster/.kubeconfig

    # Update Kube config to support unsecured connection
    cluster_port=$(./cluster/cli.sh ports ocp | tr -d '\r')
    ./cluster/kubectl.sh config set-cluster node01:8443 --server=https://127.0.0.1:$cluster_port
    ./cluster/kubectl.sh config set-cluster node01:8443 --insecure-skip-tls-verify=true
fi

echo 'Wait until all nodes are ready'
until [[ $(./cluster/kubectl.sh get nodes --no-headers | wc -l) -eq $(./cluster/kubectl.sh get nodes --no-headers | grep ' Ready' | wc -l) ]]; do
    sleep 1
done

echo 'Install Open vSwitch on nodes'
if [[ $image == $KUBERNETES_IMAGE ]]; then
    for i in $(seq 1 ${KUBEVIRT_NUM_NODES}); do
        ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" -- sudo yum install -y http://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-2.9.2-1.el7.x86_64.rpm http://cbs.centos.org/kojifiles/packages/openvswitch/2.9.2/1.el7/x86_64/openvswitch-devel-2.9.2-1.el7.x86_64.rpm http://cbs.centos.org/kojifiles/packages/dpdk/17.11/3.el7/x86_64/dpdk-17.11-3.el7.x86_64.rpm
        ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" -- sudo systemctl daemon-reload
        ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" -- sudo systemctl restart openvswitch
    done
elif [[ $image == $OPENSHIFT_IMAGE ]]; then
    ./cluster/kubectl.sh create -f _out/manifests/openshift-ovs-vsctl.yaml
    until [[ $(./cluster/kubectl.sh -n kube-system get daemonsets | grep ovs-vsctl-amd64 | awk '{ if ($3 == $4) print "0"; else print "1"}') -eq "0" ]]; do
        sleep 1
    done
fi

echo 'Install NetworkManager on nodes'
for i in $(seq 1 ${KUBEVIRT_NUM_NODES}); do
    ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" -- sudo yum install -y NetworkManager NetworkManager-ovs
    ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" -- sudo systemctl daemon-reload
    ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" -- sudo systemctl restart NetworkManager
done
