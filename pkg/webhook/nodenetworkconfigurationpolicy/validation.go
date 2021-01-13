package nodenetworkconfigurationpolicy

import (
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	shared "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

func onPolicySpecChange(policy nmstatev1beta1.NodeNetworkConfigurationPolicy, currentPolicy nmstatev1beta1.NodeNetworkConfigurationPolicy) bool {
	return !reflect.DeepEqual(policy.Spec, currentPolicy.Spec)
}

func validatePolicyNotInProgressHook(policy nmstatev1beta1.NodeNetworkConfigurationPolicy, currentPolicy nmstatev1beta1.NodeNetworkConfigurationPolicy) []metav1.StatusCause {
	causes := []metav1.StatusCause{}
	currentPolicyAvailableCondition := currentPolicy.Status.Conditions.Find(shared.NodeNetworkConfigurationPolicyConditionAvailable)

	if currentPolicyAvailableCondition == nil ||
		currentPolicyAvailableCondition.Reason == "" ||
		currentPolicyAvailableCondition.Reason == shared.NodeNetworkConfigurationPolicyConditionConfigurationProgressing {
		causes = append(causes, metav1.StatusCause{
			Message: fmt.Sprintf("policy %s is still in progress", currentPolicy.Name),
		})
	}
	return causes
}

func validatePolicyUpdateHook(cli client.Client) *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(
			validatePolicyHandler(
				cli,
				onPolicySpecChange,
				validatePolicyNotInProgressHook,
			),
		),
	}
}
