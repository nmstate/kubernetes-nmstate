/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodenetworkconfigurationpolicy

import (
	"fmt"
	"reflect"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	shared "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

func onPolicySpecChange(
	_ admissionv1.Operation,
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	currentPolicy *nmstatev1.NodeNetworkConfigurationPolicy,
) bool {
	return !reflect.DeepEqual(policy.Spec, currentPolicy.Spec)
}

func onCreate(
	operation admissionv1.Operation,
	_ *nmstatev1.NodeNetworkConfigurationPolicy,
	_ *nmstatev1.NodeNetworkConfigurationPolicy,
) bool {
	return operation == admissionv1.Create
}

func validatePolicyNotInProgressHook(
	_ *nmstatev1.NodeNetworkConfigurationPolicy,
	currentPolicy *nmstatev1.NodeNetworkConfigurationPolicy,
) []metav1.StatusCause {
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

func validatePolicyNodeSelector(
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	_ *nmstatev1.NodeNetworkConfigurationPolicy,
) []metav1.StatusCause {
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

func validatePolicyName(
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	_ *nmstatev1.NodeNetworkConfigurationPolicy,
) []metav1.StatusCause {
	causes := []metav1.StatusCause{}
	validationErrors := validation.IsValidLabelValue(policy.Name)
	if len(validationErrors) > 0 {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("invalid policy name: %q: %s", policy.Name, strings.Join(validationErrors, "; ")),
			Field:   "name",
		})
	}
	return causes
}

func validatePolicyCaptureNotModified(
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	currentPolicy *nmstatev1.NodeNetworkConfigurationPolicy,
) []metav1.StatusCause {
	causes := []metav1.StatusCause{}
	if !reflect.DeepEqual(policy.Spec.Capture, currentPolicy.Spec.Capture) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueNotSupported,
			Message: "invalid policy operation: capture field cannot be modified",
			Field:   "capture",
		})
	}
	return causes
}

func validatePolicyUpdateHook(cli client.Client) *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.MultiValidatingHandler(
			validatePolicyHandler(
				cli,
				onPolicySpecChange,
				validatePolicyNotInProgressHook,
				validatePolicyNodeSelector,
				validatePolicyCaptureNotModified,
			),
		),
	}
}

func validatePolicyCreateHook(cli client.Client) *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.MultiValidatingHandler(
			validatePolicyHandler(
				cli,
				onCreate,
				validatePolicyName,
			),
		),
	}
}
