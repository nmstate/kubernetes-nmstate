package nodenetworkconfigurationpolicy

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

func deleteConditionsHook() *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(
			mutatePolicyHandler(
				always,
				deleteConditions,
			)),
	}
}

func deleteConditionsOnNodeLabelsModifiedHook(cli client.Client) *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(
			mutateAllPoliciesHandler(cli,
				onModifiedNodeLabels,
				deleteConditions,
				setTimestampAnnotation(TimestampAllPoliciesLabelKey),
			)),
	}
}

func setConditionsUnknownHook() *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(
			mutatePolicyHandler(
				atEmptyConditions,
				setConditionsUnknown,
			)),
	}
}

func setTimestampAnnotationHook() *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(
			mutatePolicyHandler(
				always,
				setTimestampAnnotation(TimestampPolicyLabelKey),
			)),
	}
}

func always(nmstatev1beta1.NodeNetworkConfigurationPolicy) bool {
	return true
}
