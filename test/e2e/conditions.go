package e2e

import (
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
	yaml "sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

type expectedConditionsStatus struct {
	Node       string
	conditions nmstatev1alpha1.ConditionList
}

func conditionsToYaml(conditions nmstatev1alpha1.ConditionList) string {
	manifest, err := yaml.Marshal(conditions)
	if err != nil {
		panic(err)
	}
	return string(manifest)
}

func enactmentConditionsStatus(node string) nmstatev1alpha1.ConditionList {
	key := types.NamespacedName{Name: TestPolicy}
	policy := nodeNetworkConfigurationPolicy(key)
	enactmentsConditionTypes := []nmstatev1alpha1.ConditionType{
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionAvailable,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionFailing,
		nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionProgressing,
	}
	obtainedConditions := nmstatev1alpha1.ConditionList{}
	for _, enactmentsConditionType := range enactmentsConditionTypes {
		obtainedCondition := policy.Status.Enactments.FindCondition(node, enactmentsConditionType)
		obtainedConditionStatus := corev1.ConditionUnknown
		if obtainedCondition != nil {
			obtainedConditionStatus = obtainedCondition.Status
		}
		obtainedConditions = append(obtainedConditions, nmstatev1alpha1.Condition{
			Type:   enactmentsConditionType,
			Status: obtainedConditionStatus,
		})
	}
	return obtainedConditions
}

func enactmentConditionsStatusEventually(node string) AsyncAssertion {
	return Eventually(func() nmstatev1alpha1.ConditionList {
		return enactmentConditionsStatus(node)
	}, 180*time.Second, 1*time.Second)
}

func enactmentConditionsStatusConsistently(node string) AsyncAssertion {
	return Consistently(func() nmstatev1alpha1.ConditionList {
		return enactmentConditionsStatus(node)
	}, 5*time.Second, 1*time.Second)
}
