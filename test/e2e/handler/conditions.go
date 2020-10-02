package handler

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/types"

	"k8s.io/apimachinery/pkg/types"
	yaml "sigs.k8s.io/yaml"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	corev1 "k8s.io/api/core/v1"

	shared "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

type expectedConditionsStatus struct {
	Node       string
	conditions shared.ConditionList
}

func conditionsToYaml(conditions shared.ConditionList) string {
	manifest, err := yaml.Marshal(conditions)
	if err != nil {
		panic(err)
	}
	return string(manifest)
}

func nodeNetworkConfigurationEnactment(key types.NamespacedName) nmstatev1beta1.NodeNetworkConfigurationEnactment {
	enactment := nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	Eventually(func() error {
		return framework.Global.Client.Get(context.TODO(), key, &enactment)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred())
	return enactment
}

func enactmentConditionsStatus(node string, policy string) shared.ConditionList {
	enactment := nodeNetworkConfigurationEnactment(shared.EnactmentKey(node, policy))
	obtainedConditions := shared.ConditionList{}
	for _, enactmentsConditionType := range shared.NodeNetworkConfigurationEnactmentConditionTypes {
		obtainedCondition := enactment.Status.Conditions.Find(enactmentsConditionType)
		obtainedConditionStatus := corev1.ConditionUnknown
		if obtainedCondition != nil {
			obtainedConditionStatus = obtainedCondition.Status
		}
		obtainedConditions = append(obtainedConditions, shared.Condition{
			Type:   enactmentsConditionType,
			Status: obtainedConditionStatus,
		})
	}
	return obtainedConditions
}

func enactmentConditionsStatusForPolicyEventually(node string, policy string) AsyncAssertion {
	return Eventually(func() shared.ConditionList {
		return enactmentConditionsStatus(node, policy)
	}, 180*time.Second, 1*time.Second)
}

func enactmentConditionsStatusForPolicyConsistently(node string, policy string) AsyncAssertion {
	return Consistently(func() shared.ConditionList {
		return enactmentConditionsStatus(node, policy)
	}, 5*time.Second, 1*time.Second)
}

func enactmentConditionsStatusEventually(node string) AsyncAssertion {
	return enactmentConditionsStatusForPolicyEventually(node, TestPolicy)
}

func enactmentConditionsStatusConsistently(node string) AsyncAssertion {
	return enactmentConditionsStatusForPolicyConsistently(node, TestPolicy)
}

// In case a condition does not exist create with Unknonw type, this way
// is easier to just use gomega matchers to check in a homogenous way that
// condition is not present or unknown.
func policyConditionsStatus(policyName string) shared.ConditionList {
	policy := nodeNetworkConfigurationPolicy(policyName)
	obtainedConditions := shared.ConditionList{}
	for _, policyConditionType := range shared.NodeNetworkConfigurationPolicyConditionTypes {
		obtainedCondition := policy.Status.Conditions.Find(policyConditionType)
		obtainedConditionStatus := corev1.ConditionUnknown
		if obtainedCondition != nil {
			obtainedConditionStatus = obtainedCondition.Status
		}
		obtainedConditions = append(obtainedConditions, shared.Condition{
			Type:   policyConditionType,
			Status: obtainedConditionStatus,
		})
	}
	return obtainedConditions
}

func policyConditionsStatusForPolicyEventually(policy string) AsyncAssertion {
	return Eventually(func() shared.ConditionList {
		return policyConditionsStatus(policy)
	}, 480*time.Second, 1*time.Second)
}

func policyConditionsStatusForPolicyConsistently(policy string) AsyncAssertion {
	return Consistently(func() shared.ConditionList {
		return policyConditionsStatus(policy)
	}, 5*time.Second, 1*time.Second)
}

func policyConditionsStatusEventually() AsyncAssertion {
	return policyConditionsStatusForPolicyEventually(TestPolicy)
}

func policyConditionsStatusConsistently() AsyncAssertion {
	return policyConditionsStatusForPolicyConsistently(TestPolicy)
}

func containPolicyAvailable() GomegaMatcher {
	return ContainElement(shared.Condition{
		Type:   shared.NodeNetworkConfigurationPolicyConditionAvailable,
		Status: corev1.ConditionTrue,
	})
}

func containPolicyDegraded() GomegaMatcher {
	return ContainElement(shared.Condition{
		Type:   shared.NodeNetworkConfigurationPolicyConditionDegraded,
		Status: corev1.ConditionTrue,
	})
}

func waitForAvailableTestPolicy() {
	policyConditionsStatusEventually().Should(containPolicyAvailable())
}

func waitForDegradedTestPolicy() {
	policyConditionsStatusEventually().Should(containPolicyDegraded())
}

func waitForAvailablePolicy(policy string) {
	policyConditionsStatusForPolicyEventually(policy).Should(containPolicyAvailable())
}

func waitForDegradedPolicy(policy string) {
	policyConditionsStatusForPolicyEventually(policy).Should(containPolicyDegraded())
}
