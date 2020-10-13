package nodenetworkconfigurationpolicy

import (
	"strconv"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

const (
	TimestampLabelKey = "nmstate.io/webhook-mutating-timestamp"
)

func setTimestampAnnotation(policy nmstatev1beta1.NodeNetworkConfigurationPolicy) nmstatev1beta1.NodeNetworkConfigurationPolicy {
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
