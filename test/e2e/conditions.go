package e2e

import (
	"context"
	"time"

	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
	yaml "sigs.k8s.io/yaml"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

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

func nodeNetworkConfigurationEnactment(key types.NamespacedName) nmstatev1alpha1.NodeNetworkConfigurationEnactment {
	enactment := nmstatev1alpha1.NodeNetworkConfigurationEnactment{}
	Eventually(func() error {
		return framework.Global.Client.Get(context.TODO(), key, &enactment)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred())
	return enactment
}

func enactmentConditionsStatus(node string, policy string) nmstatev1alpha1.ConditionList {
	enactment := nodeNetworkConfigurationEnactment(nmstatev1alpha1.EnactmentKey(node, policy))
	obtainedConditions := nmstatev1alpha1.ConditionList{}
	for _, enactmentsConditionType := range nmstatev1alpha1.NodeNetworkConfigurationEnactmentConditionTypes {
		obtainedCondition := enactment.Status.Conditions.Find(enactmentsConditionType)
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

func enactmentConditionsStatusForPolicyEventually(node string, policy string) AsyncAssertion {
	return Eventually(func() nmstatev1alpha1.ConditionList {
		return enactmentConditionsStatus(node, policy)
	}, 180*time.Second, 1*time.Second)
}

func enactmentConditionsStatusForPolicyConsistently(node string, policy string) AsyncAssertion {
	return Consistently(func() nmstatev1alpha1.ConditionList {
		return enactmentConditionsStatus(node, policy)
	}, 5*time.Second, 1*time.Second)
}

func enactmentConditionsStatusEventually(node string) AsyncAssertion {
	return enactmentConditionsStatusForPolicyEventually(node, TestPolicy)
}

func enactmentConditionsStatusConsistently(node string) AsyncAssertion {
	return enactmentConditionsStatusForPolicyConsistently(node, TestPolicy)
}
