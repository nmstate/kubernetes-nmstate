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

package policy

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

const (
	ReadTimeout  = 180 * time.Second
	ReadInterval = 1 * time.Second
	TestPolicy   = "test-policy"
)

func EnactmentsStatusToYaml() string {
	enactmentsStatus := IndexEnactmentStatusByName()
	manifest, err := yaml.Marshal(enactmentsStatus)
	Expect(err).ToNot(HaveOccurred())
	return string(manifest)
}

func NodeNetworkConfigurationEnactment(key types.NamespacedName) nmstatev1beta1.NodeNetworkConfigurationEnactment {
	enactment := nmstatev1beta1.NodeNetworkConfigurationEnactment{}
	Eventually(func() error {
		return testenv.Client.Get(context.TODO(), key, &enactment)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred())
	return enactment
}

func IndexEnactmentStatusByName() map[string]shared.NodeNetworkConfigurationEnactmentStatus {
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

func EnactmentConditionsStatus(node, policy string) shared.ConditionList {
	return NodeNetworkConfigurationEnactment(shared.EnactmentKey(node, policy)).Status.Conditions
}

func EnactmentConditionsStatusForPolicyEventually(node string, policy string) AsyncAssertion {
	return Eventually(func() shared.ConditionList {
		return EnactmentConditionsStatus(node, policy)
	}, 180*time.Second, 1*time.Second)
}

func EnactmentConditionsStatusForPolicyConsistently(node, policy string) AsyncAssertion {
	return Consistently(func() shared.ConditionList {
		return EnactmentConditionsStatus(node, policy)
	}, 5*time.Second, 1*time.Second)
}

func EnactmentConditionsStatusEventually(node string) AsyncAssertion {
	return EnactmentConditionsStatusForPolicyEventually(node, TestPolicy)
}

func EnactmentConditionsStatusConsistently(node string) AsyncAssertion {
	return EnactmentConditionsStatusForPolicyConsistently(node, TestPolicy)
}

// Status In case a condition does not exist create with Unknown type, this way
// is easier to just use gomega matchers to check in a homogenous way that
// condition is not present or unknown.
func Status(policyName string) shared.ConditionList {
	policy := NodeNetworkConfigurationPolicy(policyName)
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

func NodeNetworkConfigurationPolicy(policyName string) nmstatev1.NodeNetworkConfigurationPolicy {
	key := types.NamespacedName{Name: policyName}
	policy := nmstatev1.NodeNetworkConfigurationPolicy{}
	EventuallyWithOffset(1, func() error {
		return testenv.Client.Get(context.TODO(), key, &policy)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred())
	return policy
}

func StatusForPolicyEventually(policy string) AsyncAssertion {
	return Eventually(func() shared.ConditionList {
		return Status(policy)
	}, 480*time.Second, 1*time.Second)
}

func StatusForPolicyConsistently(policy string) AsyncAssertion {
	return Consistently(func() shared.ConditionList {
		return Status(policy)
	}, 5*time.Second, 1*time.Second)
}

func StatusEventually() AsyncAssertion {
	return StatusForPolicyEventually(TestPolicy)
}

func StatusConsistently() AsyncAssertion {
	return StatusForPolicyConsistently(TestPolicy)
}

func ContainPolicyAvailable() gomegatypes.GomegaMatcher {
	return ContainElement(MatchFields(IgnoreExtras, Fields{
		"Type":    Equal(shared.NodeNetworkConfigurationPolicyConditionAvailable),
		"Status":  Equal(corev1.ConditionTrue),
		"Reason":  Not(BeEmpty()),
		"Message": Not(BeEmpty()),
	}))
}

func ContainPolicyDegraded() gomegatypes.GomegaMatcher {
	return ContainElement(MatchFields(IgnoreExtras, Fields{
		"Type":    Equal(shared.NodeNetworkConfigurationPolicyConditionDegraded),
		"Status":  Equal(corev1.ConditionTrue),
		"Reason":  Not(BeEmpty()),
		"Message": Not(BeEmpty()),
	}))
}

func WaitForPolicyTransitionUpdateWithTime(policy string, applyTime time.Time) {
	// the k8s times at status are rounded to seconds
	roundedApplyTime := metav1.NewTime(applyTime).Rfc3339Copy().Time
	EventuallyWithOffset(1, func() time.Time {
		availableCondition := Status(policy).Find(shared.NodeNetworkConfigurationPolicyConditionAvailable)
		return availableCondition.LastTransitionTime.Time
	}, 4*time.Minute, 5*time.Second).Should(BeTemporally(">=", roundedApplyTime),
		fmt.Sprintf("Policy %s should have updated transition time", policy))
}

func WaitForPolicyTransitionUpdate(policy string) {
	WaitForPolicyTransitionUpdateWithTime(policy, time.Now())
}

func WaitForAvailableTestPolicy() {
	WaitForAvailablePolicy(TestPolicy)
}

func WaitForDegradedTestPolicy() {
	WaitForDegradedPolicy(TestPolicy)
}

func WaitForAvailablePolicy(policy string) {
	WaitForPolicy(policy, ContainPolicyAvailable())
}

func WaitForDegradedPolicy(policy string) {
	WaitForPolicy(policy, ContainPolicyDegraded())
}

func WaitForPolicy(policy string, matcher gomegatypes.GomegaMatcher) {
	StatusForPolicyEventually(policy).
		Should(
			SatisfyAny(ContainPolicyAvailable(), ContainPolicyDegraded()),
			func() string {
				return fmt.Sprintf("should reach terminal status at NNCP '%s', \n current enactments statuses:\n%s",
					policy, EnactmentsStatusToYaml())
			},
		)
	Expect(Status(policy)).To(matcher, "should reach expected status at NNCP '%s', \n current enactments statuses:\n%s",
		policy, EnactmentsStatusToYaml())
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

func MatchConditionsFrom(conditionsSetter func(*shared.ConditionList, string)) gomegatypes.GomegaMatcher {
	expectedConditions := shared.ConditionList{}
	conditionsSetter(&expectedConditions, "")
	expectedConditions = filterOutMessageAndTimestampFromConditions(expectedConditions)
	return WithTransform(filterOutMessageAndTimestampFromConditions, ConsistOf(expectedConditions))
}
