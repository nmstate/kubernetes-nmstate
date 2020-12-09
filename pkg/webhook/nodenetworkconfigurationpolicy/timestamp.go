package nodenetworkconfigurationpolicy

import (
	"strconv"
	"time"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

const (
	TimestampPolicyLabelKey      = "nmstate.io/webhook-mutating-policy-timestamp"
	TimestampAllPoliciesLabelKey = "nmstate.io/webhook-mutating-all-policies-timestamp"
)

func setTimestampAnnotation(timestampKey string) func(nmstatev1beta1.NodeNetworkConfigurationPolicy) nmstatev1beta1.NodeNetworkConfigurationPolicy {
	return func(policy nmstatev1beta1.NodeNetworkConfigurationPolicy) nmstatev1beta1.NodeNetworkConfigurationPolicy {
		value := strconv.FormatInt(time.Now().UnixNano(), 10)
		if policy.ObjectMeta.Annotations == nil {
			policy.ObjectMeta.Annotations = map[string]string{}
		}
		policy.ObjectMeta.Annotations[timestampKey] = value
		return policy
	}
}
