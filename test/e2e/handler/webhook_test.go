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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/util/retry"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nncpwebhook "github.com/nmstate/kubernetes-nmstate/pkg/webhook/nodenetworkconfigurationpolicy"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

// We just check the labe at CREATE/UPDATE events since mutated data is already
// check at unit test.
var _ = Describe("Mutating Admission Webhook", func() {
	Context("when policy is created", func() {
		BeforeEach(func() {
			// Make sure test policy is not there so
			// we exercise CREATE event
			resetDesiredStateForNodes()
			updateDesiredStateAndWait(linuxBrUp(bridge1))
		})
		AfterEach(func() {
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			resetDesiredStateForNodes()
		})

		It("should have an annotation with mutation timestamp", func() {
			policy := nodeNetworkConfigurationPolicy(TestPolicy)
			Expect(policy.ObjectMeta.Annotations).To(HaveKey(nncpwebhook.TimestampLabelKey))
		})
		Context("and we updated it", func() {
			var (
				oldPolicy nmstatev1.NodeNetworkConfigurationPolicy
			)
			BeforeEach(func() {
				oldPolicy = nodeNetworkConfigurationPolicy(TestPolicy)
				updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			})
			It("should have an annotation with newer mutation timestamp", func() {
				newPolicy := nodeNetworkConfigurationPolicy(TestPolicy)
				Expect(newPolicy.ObjectMeta.Annotations).To(HaveKey(nncpwebhook.TimestampLabelKey))

				oldAnnotation := oldPolicy.ObjectMeta.Annotations[nncpwebhook.TimestampLabelKey]
				oldConditionsMutation, err := strconv.ParseInt(oldAnnotation, 10, 64)
				Expect(err).ToNot(HaveOccurred())
				newAnnotation := newPolicy.ObjectMeta.Annotations[nncpwebhook.TimestampLabelKey]
				newConditionsMutation, err := strconv.ParseInt(newAnnotation, 10, 64)
				Expect(err).ToNot(HaveOccurred())

				Expect(newConditionsMutation).To(BeNumerically(">", oldConditionsMutation), "mutation timestamp not updated")
			})
		})
	})
})

var _ = Describe("Validation Admission Webhook", func() {
	Context("When a policy is created and progressing", func() {
		BeforeEach(func() {
			By("Creating a policy without waiting for it to be available")
			updateDesiredState(linuxBrUp(bridge1))
		})
		AfterEach(func() {
			waitForAvailablePolicy(TestPolicy)
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			resetDesiredStateForNodes()
		})
		It("Should deny updating rolled out policy when it's in progress", func() {
			Byf("Updating the policy %s", TestPolicy)
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				return setDesiredStateWithPolicyAndNodeSelector(TestPolicy, linuxBrUpNoPorts(bridge1), map[string]string{})
			})
			Expect(err).To(MatchError("admission webhook \"nodenetworkconfigurationpolicies-update-validate.nmstate.io\" denied the request: failed to admit NodeNetworkConfigurationPolicy test-policy: message: policy test-policy is still in progress. "))
		})
	})
	Context("When a policy with too long name is created", func() {
		const tooLongName = "this-is-longer-than-sixty-three-characters-hostnames-bar-bar.com"
		It("Should deny creating policy with name longer than 63 characters", func() {
			policy := nmstatev1.NodeNetworkConfigurationPolicy{}
			policy.Name = tooLongName
			err := testenv.Client.Create(context.TODO(), &policy)
			Expect(err).To(MatchError("admission webhook \"nodenetworkconfigurationpolicies-create-validate.nmstate.io\" denied the request: failed to admit NodeNetworkConfigurationPolicy this-is-longer-than-sixty-three-characters-hostnames-bar-bar.com: message: invalid policy name: \"this-is-longer-than-sixty-three-characters-hostnames-bar-bar.com\": must be no more than 63 characters. "))
		})
	})
	Context("When a policy capture field is updated", func() {
		BeforeEach(func() {
			By("Create a policy without capture field")
			updateDesiredStateAndWait(linuxBrUp(bridge1))
		})
		It("should deny creating the capture field", func() {
			By("Add capture field to the NNCP")
			capture := map[string]string{"default-gw": `routes.running.destination=="0.0.0.0/0"`}
			err := setDesiredStateWithPolicyAndCaptureAndNodeSelector(TestPolicy, linuxBrUpNoPorts(bridge1), capture, map[string]string{})
			Expect(err).To(MatchError(`admission webhook "nodenetworkconfigurationpolicies-update-validate.nmstate.io" denied the request: failed to admit NodeNetworkConfigurationPolicy test-policy: message: invalid policy operation: capture field cannot be modified. `))
		})
		AfterEach(func() {
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			resetDesiredStateForNodes()
		})
	})
})
