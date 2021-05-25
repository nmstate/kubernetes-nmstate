package nodenetworkconfigurationpolicy

import (
	"fmt"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
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

func validatePolicyNodeSelector(policy nmstatev1beta1.NodeNetworkConfigurationPolicy, currentPolicy nmstatev1beta1.NodeNetworkConfigurationPolicy) []metav1.StatusCause {
	causes := []metav1.StatusCause{}
	nodeSelector := policy.Spec.NodeSelector
	if nodeSelector == nil {
		return causes
	}
	for labelKey, labelValue := range nodeSelector {
		validationErrors := validation.IsQualifiedName(labelKey)
		if len(validationErrors) > 0 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("invalid label key: %q: %s", labelKey, strings.Join(validationErrors, "; ")),
				Field:   "spec.nodeSelector",
			})
		}
		validationErrors = validation.IsValidLabelValue(labelValue)
		if len(validationErrors) > 0 {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("invalid label value: %q: at key: %q: %s", labelValue, labelKey, strings.Join(validationErrors, "; ")),
				Field:   "spec.nodeSelector",
			})
		}
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
				validatePolicyNodeSelector,
			),
		),
	}
}
