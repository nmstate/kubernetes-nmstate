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

package handler

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

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

func enactmentConditionsStatus(node, policy string) shared.ConditionList {
	return nodeNetworkConfigurationEnactment(shared.EnactmentKey(node, policy)).Status.Conditions
}

func enactmentConditionsStatusForPolicyEventually(node, policy string) AsyncAssertion {
	return Eventually(func() shared.ConditionList {
		return enactmentConditionsStatus(node, policy)
	}, 180*time.Second, 1*time.Second)
}

func enactmentConditionsStatusForPolicyConsistently(node, policy string) AsyncAssertion {
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

func policyConditionsStatusConsistently() AsyncAssertion {
	return policyConditionsStatusForPolicyConsistently(TestPolicy)
}

func containPolicyAvailable() gomegatypes.GomegaMatcher {
	return ContainElement(MatchFields(IgnoreExtras, Fields{
		"Type":    Equal(shared.NodeNetworkConfigurationPolicyConditionAvailable),
		"Status":  Equal(corev1.ConditionTrue),
		"Reason":  Not(BeEmpty()),
		"Message": Not(BeEmpty()),
	}))
}

func containPolicyDegraded() gomegatypes.GomegaMatcher {
	return ContainElement(MatchFields(IgnoreExtras, Fields{
		"Type":    Equal(shared.NodeNetworkConfigurationPolicyConditionDegraded),
		"Status":  Equal(corev1.ConditionTrue),
		"Reason":  Not(BeEmpty()),
		"Message": Not(BeEmpty()),
	}))
}

func waitForPolicyTransitionUpdate(policy string) {
	now := time.Now()
	EventuallyWithOffset(1, func() time.Time {
		availableCondition := policyConditionsStatus(policy).Find(shared.NodeNetworkConfigurationPolicyConditionAvailable)
		return availableCondition.LastTransitionTime.Time
	}, 4*time.Minute, 5*time.Second).Should(BeTemporally(">=", now), fmt.Sprintf("Policy %s should have updated transition time", policy))
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

func waitForPolicy(policy string, matcher gomegatypes.GomegaMatcher) {
	policyConditionsStatusForPolicyEventually(policy).
		Should(
			matcher,
			func() string {
				return fmt.Sprintf("should reach expected status at NNCP '%s', \n current enactments statuses:\n%s", policy, enactmentsStatusToYaml())
			},
		)
}

func filterOutMessageAndTimestampFromConditions(conditions shared.ConditionList) shared.ConditionList {
	modifiedConditions := shared.ConditionList{}
	for _, condition := range conditions {
		modifiedConditions = append(modifiedConditions, shared.Condition{
			Type:   condition.Type,
			Status: condition.Status,
			Reason: condition.Reason,
		})
	}
	return modifiedConditions
}

func matchConditionsFrom(conditionsSetter func(*shared.ConditionList, string)) gomegatypes.GomegaMatcher {
	expectedConditions := shared.ConditionList{}
	conditionsSetter(&expectedConditions, "")
	expectedConditions = filterOutMessageAndTimestampFromConditions(expectedConditions)
	return WithTransform(filterOutMessageAndTimestampFromConditions, ConsistOf(expectedConditions))
}
