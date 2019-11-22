package e2e

import (
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
	yaml "sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

func conditionsToYaml(conditions nmstatev1alpha1.ConditionList) string {
	manifest, err := yaml.Marshal(conditions)
	if err != nil {
		panic(err)
	}
	return string(manifest)
}
func checkEnactmentConditionsStatus(node string, expectedConditions []nmstatev1alpha1.Condition) bool {
	key := types.NamespacedName{Name: TestPolicy}
	policy := nodeNetworkConfigurationPolicy(key)
	for _, expectedCondition := range expectedConditions {
		obtainedCondition := policy.FindEnactmentCondition(node, expectedCondition.Type)
		obtainedConditionStatus := corev1.ConditionUnknown
		if obtainedCondition != nil {
			obtainedConditionStatus = obtainedCondition.Status
		}
		//TODO: Add context info to debug test failures
		if obtainedConditionStatus != expectedCondition.Status {
			return false
		}
	}
	return true
}

func checkEnactmentConditionsStatusEventually(node string, expectedConditions []nmstatev1alpha1.Condition) {
	Eventually(func() bool {
		return checkEnactmentConditionsStatus(node, expectedConditions)
	}, 180*time.Second, 1*time.Second).Should(BeTrue())
}

func checkEnactmentConditionsStatusConsistently(node string, expectedConditions []nmstatev1alpha1.Condition) {
	Consistently(func() bool {
		return checkEnactmentConditionsStatus(node, expectedConditions)
	}, 5*time.Second, 1*time.Second).Should(BeTrue())
}
