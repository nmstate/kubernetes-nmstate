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

package conditions

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	shared "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

var _ = Describe("MarkInterrupted", func() {
	It("sets Pending, clears Progressing, and zeroes the retry count", func() {
		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1beta1.GroupVersion,
			&nmstatev1beta1.NodeNetworkConfigurationEnactment{})
		nnce := nmstatev1beta1.NodeNetworkConfigurationEnactment{
			ObjectMeta: metav1.ObjectMeta{Name: "node01.test-policy"},
			Status: shared.NodeNetworkConfigurationEnactmentStatus{
				PolicyGeneration: 3,
				RetryCount:       map[string]int{"3": 2},
			},
		}
		SetProgressing(&nnce.Status.Conditions, "applying")

		clb := fake.ClientBuilder{}
		clb.WithScheme(s)
		clb.WithRuntimeObjects(&nnce)
		clb.WithStatusSubresource(&nnce)
		cl := clb.Build()

		Expect(MarkInterrupted(context.TODO(), cl,
			types.NamespacedName{Name: "node01.test-policy"}, "3")).To(Succeed())

		updated := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
		Expect(cl.Get(context.TODO(), types.NamespacedName{Name: "node01.test-policy"}, updated)).To(Succeed())
		pendingCondition := updated.Status.Conditions.Find(shared.NodeNetworkConfigurationEnactmentConditionPending)
		Expect(pendingCondition).ToNot(BeNil())
		Expect(pendingCondition.Status).To(Equal(corev1.ConditionTrue))
		Expect(pendingCondition.Message).To(ContainSubstring("interrupted by handler restart"))
		Expect(pendingCondition.Reason).To(Equal(shared.NodeNetworkConfigurationEnactmentConditionConfigurationInterrupted),
			"a restart-interrupted enactment must not reuse the MaxUnavailableLimitReached reason")
		progressingCondition := updated.Status.Conditions.Find(shared.NodeNetworkConfigurationEnactmentConditionProgressing)
		Expect(progressingCondition.Status).To(Equal(corev1.ConditionFalse))
		Expect(progressingCondition.Reason).To(Equal(shared.NodeNetworkConfigurationEnactmentConditionConfigurationInterrupted))
		Expect(updated.Status.RetryCount["3"]).To(Equal(0))
	})
})
