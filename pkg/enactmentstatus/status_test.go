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

package enactmentstatus

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func conditionList(conditions ...nmstate.Condition) *nmstate.ConditionList {
	cl := nmstate.ConditionList(conditions)
	return &cl
}

func condition(condType nmstate.ConditionType, status corev1.ConditionStatus) nmstate.Condition {
	return nmstate.Condition{
		Type:   condType,
		Status: status,
	}
}

var _ = Describe("IsAvailable", func() {
	It("should return false for empty conditions", func() {
		conditions := conditionList()
		Expect(IsAvailable(conditions)).To(BeFalse())
	})
	It("should return true when Available is True", func() {
		conditions := conditionList(
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionAvailable, corev1.ConditionTrue),
		)
		Expect(IsAvailable(conditions)).To(BeTrue())
	})
	It("should return false when Available is False", func() {
		conditions := conditionList(
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionAvailable, corev1.ConditionFalse),
		)
		Expect(IsAvailable(conditions)).To(BeFalse())
	})
	It("should return false when Available is Unknown", func() {
		conditions := conditionList(
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionAvailable, corev1.ConditionUnknown),
		)
		Expect(IsAvailable(conditions)).To(BeFalse())
	})
	It("should return false when only other conditions are present", func() {
		conditions := conditionList(
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionFailing, corev1.ConditionTrue),
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing, corev1.ConditionFalse),
		)
		Expect(IsAvailable(conditions)).To(BeFalse())
	})
})

var _ = Describe("IsProgressing", func() {
	It("should return false for empty conditions", func() {
		conditions := conditionList()
		Expect(IsProgressing(conditions)).To(BeFalse())
	})
	It("should return true when Progressing is True", func() {
		conditions := conditionList(
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing, corev1.ConditionTrue),
		)
		Expect(IsProgressing(conditions)).To(BeTrue())
	})
	It("should return false when Progressing is False", func() {
		conditions := conditionList(
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing, corev1.ConditionFalse),
		)
		Expect(IsProgressing(conditions)).To(BeFalse())
	})
	It("should return false when Progressing is Unknown", func() {
		conditions := conditionList(
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing, corev1.ConditionUnknown),
		)
		Expect(IsProgressing(conditions)).To(BeFalse())
	})
})

var _ = Describe("IsRetrying", func() {
	It("should return false for empty conditions", func() {
		conditions := conditionList()
		Expect(IsRetrying(conditions)).To(BeFalse())
	})
	It("should return true when both Failing and Progressing are True", func() {
		conditions := conditionList(
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionFailing, corev1.ConditionTrue),
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing, corev1.ConditionTrue),
		)
		Expect(IsRetrying(conditions)).To(BeTrue())
	})
	It("should return false when Failing is True but Progressing is False", func() {
		conditions := conditionList(
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionFailing, corev1.ConditionTrue),
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing, corev1.ConditionFalse),
		)
		Expect(IsRetrying(conditions)).To(BeFalse())
	})
	It("should return false when only Failing is True", func() {
		conditions := conditionList(
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionFailing, corev1.ConditionTrue),
		)
		Expect(IsRetrying(conditions)).To(BeFalse())
	})
	It("should return false when only Progressing is True", func() {
		conditions := conditionList(
			condition(nmstate.NodeNetworkConfigurationEnactmentConditionProgressing, corev1.ConditionTrue),
		)
		Expect(IsRetrying(conditions)).To(BeFalse())
	})
})
