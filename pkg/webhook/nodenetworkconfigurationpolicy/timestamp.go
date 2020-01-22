package nodenetworkconfigurationpolicy

import (
	"strconv"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

const (
	TimestampLabelKey = "nmstate.io/webhook-mutating-timestamp"
)

func setTimestampAnnotation(policy nmstatev1alpha1.NodeNetworkConfigurationPolicy) nmstatev1alpha1.NodeNetworkConfigurationPolicy {
	value := strconv.FormatInt(time.Now().UnixNano(), 10)
	if policy.ObjectMeta.Annotations == nil {
		policy.ObjectMeta.Annotations = map[string]string{}
	}
	policy.ObjectMeta.Annotations[TimestampLabelKey] = value
	return policy
}

func setTimestampAnnotationHook() *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(
			mutatePolicyHandler(
				always,
				setTimestampAnnotation,
			)),
	}
}
