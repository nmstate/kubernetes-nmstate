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
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

func deleteConditions(policy *nmstatev1.NodeNetworkConfigurationPolicy) {
	policy.Status.Conditions = shared.ConditionList{}
}

func setConditionsUnknown(policy *nmstatev1.NodeNetworkConfigurationPolicy) {
	unknownConditions := shared.ConditionList{}
	for _, conditionType := range shared.NodeNetworkConfigurationPolicyConditionTypes {
		unknownConditions.Set(
			conditionType,
			corev1.ConditionUnknown,
			"", "")
	}
	policy.Status.Conditions = unknownConditions
}

func atEmptyConditions(policy *nmstatev1.NodeNetworkConfigurationPolicy) bool {
	return len(policy.Status.Conditions) == 0
}

func deleteConditionsHook() *webhook.Admission {
	return &webhook.Admission{
		Handler: mutatePolicyHandler(
			always,
			deleteConditions,
		),
	}
}

func setConditionsUnknownHook() *webhook.Admission {
	return &webhook.Admission{
		Handler: mutatePolicyHandler(
			atEmptyConditions,
			setConditionsUnknown,
		),
	}
}
