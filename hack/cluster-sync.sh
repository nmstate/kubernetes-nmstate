#!/bin/bash -ex

${KUBECTL} apply -f deploy/crds/nmstate_v1alpha1_nodenetworkstate_crd.yaml
${KUBECTL} apply -f deploy/crds/nmstate_v1alpha1_nodenetworkconfigurationpolicy_crd.yaml
${KUBECTL} delete --ignore-not-found -f ${local_handler_manifest}
# Set debug verbosity level for logs when using cluster-sync
sed "s#--v=production#--v=debug#" $(local_handler_manifest) | $(KUBECTL) create -f -

desiredNumberScheduled="$(${KUBECTL} get daemonset -n nmstate nmstate-handler -o=jsonpath='{.status.desiredNumberScheduled}')"
for i in {60..0}; do
	if [ $desiredNumberScheduled == "$(${KUBECTL} get daemonset -n nmstate nmstate-handler -o=jsonpath='{.status.numberAvailable}')" ]; then
		echo "nmstate-handler DS is ready"
		break
	fi

	if [ $i -eq 0 ]; then
		echo "nmstate-handler DS haven't turned ready within the given timeout"
		exit 1
	fi

	sleep 5;
done
