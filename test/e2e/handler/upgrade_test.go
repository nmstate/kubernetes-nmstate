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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/api/v1alpha1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

var _ = Describe("NodeNetworkConfigurationPolicy upgrade", func() {
	Context("when v1alpha1 is populated", func() {
		BeforeEach(func() {
			maxUnavailableIntOrString := intstr.FromString(maxUnavailable)
			policy := nmstatev1alpha1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: TestPolicy,
				},
				Spec: nmstate.NodeNetworkConfigurationPolicySpec{
					DesiredState:   linuxBrUp(bridge1),
					NodeSelector:   map[string]string{"node-role.kubernetes.io/worker": ""},
					MaxUnavailable: &maxUnavailableIntOrString,
				},
			}
			Expect(testenv.Client.Create(context.TODO(), &policy)).To(Succeed(), "should success creating a v1alpha1 nncp")
		})
		AfterEach(func() {
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			resetDesiredStateForNodes()
		})
		It("should be stored as v1 and end with available state", func() {
			waitForAvailableTestPolicy()
		})
	})

	Context("when v1beta1 is populated", func() {
		BeforeEach(func() {
			maxUnavailableIntOrString := intstr.FromString(maxUnavailable)
			policy := nmstatev1beta1.NodeNetworkConfigurationPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: TestPolicy,
				},
				Spec: nmstate.NodeNetworkConfigurationPolicySpec{
					DesiredState:   linuxBrUp(bridge1),
					NodeSelector:   map[string]string{"node-role.kubernetes.io/worker": ""},
					MaxUnavailable: &maxUnavailableIntOrString,
				},
			}
			Expect(testenv.Client.Create(context.TODO(), &policy)).To(Succeed(), "should success creating a v1beta1 nncp")
		})
		AfterEach(func() {
			updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			resetDesiredStateForNodes()
		})
		It("should be stored as v1 and end with available state", func() {
			waitForAvailableTestPolicy()
		})
	})
})
