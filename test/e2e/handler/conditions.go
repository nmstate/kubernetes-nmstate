package handler

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	. "github.com/onsi/gomega/types"

	"k8s.io/apimachinery/pkg/types"
	yaml "sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"

	shared "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
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

func enactmentsStatusToYaml() string {
	enactmentsStatus := indexEnactmentStatusByName()
	manifest, err := yaml.Marshal(enactmentsStatus)
	Expect(err).ToNot(HaveOccurred())
	return string(manifest)
}

func nodeNetworkConfigurationEnactment(key types.NamespacedName) nmstatev1beta1.NodeNetworkConfigurationEnactment {
	enactment := nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	Eventually(func() error {
		return testenv.Client.Get(context.TODO(), key, &enactment)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred())
	return enactment
}

func indexEnactmentStatusByName() map[string]shared.NodeNetworkConfigurationEnactmentStatus {
	enactmentList := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
	Eventually(func() error {
		return testenv.Client.List(context.TODO(), &enactmentList)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred())
	enactmentStatusByName := map[string]shared.NodeNetworkConfigurationEnactmentStatus{}
	for _, enactment := range enactmentList.Items {
		enactmentStatusByName[enactment.Name] = enactment.Status
	}
	return enactmentStatusByName
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

// In case a condition does not exist create with Unknown type, this way
// is easier to just use gomega matchers to check in a homogenous way that
// condition is not present or unknown.
func policyConditionsStatus(policyName string) shared.ConditionList {
	policy := nodeNetworkConfigurationPolicy(policyName)
	conditions := shared.ConditionList{}
	for _, policyConditionType := range shared.NodeNetworkConfigurationPolicyConditionTypes {
		condition := policy.Status.Conditions.Find(policyConditionType)
		if condition == nil {
			condition = &shared.Condition{
				Type:   policyConditionType,
				Status: corev1.ConditionUnknown,
			}
		}
		conditions = append(conditions, *condition)
	}
	return conditions
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
	return ContainElement(MatchFields(IgnoreExtras, Fields{
		"Type":    Equal(shared.NodeNetworkConfigurationPolicyConditionAvailable),
		"Status":  Equal(corev1.ConditionTrue),
		"Reason":  Not(BeEmpty()),
		"Message": Not(BeEmpty()),
	}))
}

func containPolicyDegraded() GomegaMatcher {
	return ContainElement(MatchFields(IgnoreExtras, Fields{
		"Type":    Equal(shared.NodeNetworkConfigurationPolicyConditionDegraded),
		"Status":  Equal(corev1.ConditionTrue),
		"Reason":  Not(BeEmpty()),
		"Message": Not(BeEmpty()),
	}))
}

func waitForAvailableTestPolicy() {
	waitForAvailablePolicy(TestPolicy)
}

func waitForDegradedTestPolicy() {
	waitForDegradedPolicy(TestPolicy)
}

func waitForAvailablePolicy(policy string) {
	waitForPolicy(policy, containPolicyAvailable())
}

func waitForDegradedPolicy(policy string) {
	waitForPolicy(policy, containPolicyDegraded())
}

func waitForPolicy(policy string, matcher GomegaMatcher) {
	policyConditionsStatusForPolicyEventually(policy).Should(matcher, "should reach expected status at NNCP '%s', \n current enactments statuses:\n%s", policy, enactmentsStatusToYaml())
}
