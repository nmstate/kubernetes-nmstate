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

package client

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
)

var _ = Describe("SetNodeNetworkStateQueryConditions", func() {
	expectConditions := func(conditions shared.ConditionList, available, failing corev1.ConditionStatus,
		reason shared.ConditionReason) {
		availableCondition := conditions.Find(shared.NodeNetworkStateConditionAvailable)
		ExpectWithOffset(1, availableCondition).ToNot(BeNil())
		ExpectWithOffset(1, availableCondition.Status).To(Equal(available))
		ExpectWithOffset(1, availableCondition.Reason).To(Equal(reason))
		failingCondition := conditions.Find(shared.NodeNetworkStateConditionFailing)
		ExpectWithOffset(1, failingCondition).ToNot(BeNil())
		ExpectWithOffset(1, failingCondition.Status).To(Equal(failing))
		ExpectWithOffset(1, failingCondition.Reason).To(Equal(reason))
	}

	It("should mark the state available on success", func() {
		conditions := shared.ConditionList{}
		Expect(SetNodeNetworkStateQueryConditions(&conditions,
			shared.NodeNetworkStateConditionQuerySucceeded, QuerySucceededMessage)).To(BeTrue())
		expectConditions(conditions, corev1.ConditionTrue, corev1.ConditionFalse,
			shared.NodeNetworkStateConditionQuerySucceeded)
	})

	It("should mark the state failing on any other reason", func() {
		conditions := shared.ConditionList{}
		Expect(SetNodeNetworkStateQueryConditions(&conditions,
			shared.NodeNetworkStateConditionNetworkManagerUnresponsive, "boom")).To(BeTrue())
		expectConditions(conditions, corev1.ConditionFalse, corev1.ConditionTrue,
			shared.NodeNetworkStateConditionNetworkManagerUnresponsive)
	})

	It("should report no change and keep heartbeat untouched when nothing changes", func() {
		conditions := shared.ConditionList{}
		SetNodeNetworkStateQueryConditions(&conditions, shared.NodeNetworkStateConditionQuerySucceeded, QuerySucceededMessage)
		heartbeat := conditions.Find(shared.NodeNetworkStateConditionAvailable).LastHeartbeatTime

		Expect(SetNodeNetworkStateQueryConditions(&conditions,
			shared.NodeNetworkStateConditionQuerySucceeded, QuerySucceededMessage)).To(BeFalse())
		Expect(conditions.Find(shared.NodeNetworkStateConditionAvailable).LastHeartbeatTime).To(Equal(heartbeat))
	})

	It("should report a change when the message changes", func() {
		conditions := shared.ConditionList{}
		SetNodeNetworkStateQueryConditions(&conditions, shared.NodeNetworkStateConditionQueryFailed, "first")
		Expect(SetNodeNetworkStateQueryConditions(&conditions,
			shared.NodeNetworkStateConditionQueryFailed, "second")).To(BeTrue())
		Expect(conditions.Find(shared.NodeNetworkStateConditionFailing).Message).To(Equal("second"))
	})
})
