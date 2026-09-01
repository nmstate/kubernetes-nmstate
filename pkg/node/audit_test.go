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

package node

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

func auditPolicy(generation int64, count int, lastUpdate *metav1.Time) *nmstatev1.NodeNetworkConfigurationPolicy {
	return &nmstatev1.NodeNetworkConfigurationPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "test-policy", Generation: generation},
		Status: shared.NodeNetworkConfigurationPolicyStatus{
			UnavailableNodeCountMap:        map[string]int{"2": count},
			LastUnavailableNodeCountUpdate: lastUpdate,
		},
	}
}

func auditEnactment(
	name string,
	policyGeneration int64,
	progressing corev1.ConditionStatus,
	heartbeatAge time.Duration,
) *nmstatev1beta1.NodeNetworkConfigurationEnactment {
	e := &nmstatev1beta1.NodeNetworkConfigurationEnactment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{shared.EnactmentPolicyLabel: "test-policy"},
		},
		Status: shared.NodeNetworkConfigurationEnactmentStatus{
			PolicyGeneration: policyGeneration,
			Conditions: shared.ConditionList{
				shared.Condition{
					Type:              shared.NodeNetworkConfigurationEnactmentConditionProgressing,
					Status:            progressing,
					LastHeartbeatTime: metav1.Time{Time: time.Now().Add(-heartbeatAge)},
				},
			},
		},
	}
	return e
}

var _ = Describe("AuditUnavailableSlots", func() {
	var (
		policyKey = types.NamespacedName{Name: "test-policy"}
		oldUpdate = metav1.Time{Time: time.Now().Add(-5 * time.Minute)}
	)

	buildClient := func() *fake.ClientBuilder {
		s := scheme.Scheme
		s.AddKnownTypes(nmstatev1.GroupVersion,
			&nmstatev1.NodeNetworkConfigurationPolicy{})
		s.AddKnownTypes(nmstatev1beta1.GroupVersion,
			&nmstatev1beta1.NodeNetworkConfigurationEnactment{},
			&nmstatev1beta1.NodeNetworkConfigurationEnactmentList{})
		clb := fake.ClientBuilder{}
		clb.WithScheme(s)
		return &clb
	}

	It("repairs a ghost slot (count>0, no live holders)", func() {
		policy := auditPolicy(2, 1, &oldUpdate)
		clb := buildClient()
		clb.WithRuntimeObjects(policy)
		clb.WithStatusSubresource(policy)
		cl := clb.Build()

		repaired, err := AuditUnavailableSlots(context.TODO(), cl, cl, policyKey, DefaultStaleEnactmentThreshold)
		Expect(err).ToNot(HaveOccurred())
		Expect(repaired).To(BeTrue())

		updated := &nmstatev1.NodeNetworkConfigurationPolicy{}
		Expect(cl.Get(context.TODO(), policyKey, updated)).To(Succeed())
		Expect(updated.Status.UnavailableNodeCountMap["2"]).To(Equal(0))
		Expect(updated.Status.LastUnavailableNodeCountUpdate).ToNot(BeNil())
	})

	It("does not repair when a fresh Progressing holder exists", func() {
		policy := auditPolicy(2, 1, &oldUpdate)
		holder := auditEnactment("node01.test-policy", 2, corev1.ConditionTrue, time.Minute)
		clb := buildClient()
		clb.WithRuntimeObjects(policy, holder)
		clb.WithStatusSubresource(policy)
		cl := clb.Build()

		repaired, err := AuditUnavailableSlots(context.TODO(), cl, cl, policyKey, DefaultStaleEnactmentThreshold)
		Expect(err).ToNot(HaveOccurred())
		Expect(repaired).To(BeFalse())

		updated := &nmstatev1.NodeNetworkConfigurationPolicy{}
		Expect(cl.Get(context.TODO(), policyKey, updated)).To(Succeed())
		Expect(updated.Status.UnavailableNodeCountMap["2"]).To(Equal(1))
	})

	It("repairs when the only Progressing holder is stale", func() {
		policy := auditPolicy(2, 1, &oldUpdate)
		dead := auditEnactment("node01.test-policy", 2, corev1.ConditionTrue, DefaultStaleEnactmentThreshold+time.Minute)
		clb := buildClient()
		clb.WithRuntimeObjects(policy, dead)
		clb.WithStatusSubresource(policy)
		cl := clb.Build()

		repaired, err := AuditUnavailableSlots(context.TODO(), cl, cl, policyKey, DefaultStaleEnactmentThreshold)
		Expect(err).ToNot(HaveOccurred())
		Expect(repaired).To(BeTrue())

		updated := &nmstatev1.NodeNetworkConfigurationPolicy{}
		Expect(cl.Get(context.TODO(), policyKey, updated)).To(Succeed())
		Expect(updated.Status.UnavailableNodeCountMap["2"]).To(Equal(0))
	})

	It("defers inside the grace window", func() {
		recent := metav1.Time{Time: time.Now().Add(-5 * time.Second)}
		policy := auditPolicy(2, 1, &recent)
		clb := buildClient()
		clb.WithRuntimeObjects(policy)
		clb.WithStatusSubresource(policy)
		cl := clb.Build()

		repaired, err := AuditUnavailableSlots(context.TODO(), cl, cl, policyKey, DefaultStaleEnactmentThreshold)
		Expect(err).ToNot(HaveOccurred())
		Expect(repaired).To(BeFalse())
	})

	It("audits when LastUnavailableNodeCountUpdate is nil", func() {
		policy := auditPolicy(2, 1, nil)
		clb := buildClient()
		clb.WithRuntimeObjects(policy)
		clb.WithStatusSubresource(policy)
		cl := clb.Build()

		repaired, err := AuditUnavailableSlots(context.TODO(), cl, cl, policyKey, DefaultStaleEnactmentThreshold)
		Expect(err).ToNot(HaveOccurred())
		Expect(repaired).To(BeTrue())
	})

	It("ignores enactments from other generations", func() {
		policy := auditPolicy(2, 1, &oldUpdate)
		oldGen := auditEnactment("node01.test-policy", 1, corev1.ConditionTrue, time.Minute)
		clb := buildClient()
		clb.WithRuntimeObjects(policy, oldGen)
		clb.WithStatusSubresource(policy)
		cl := clb.Build()

		repaired, err := AuditUnavailableSlots(context.TODO(), cl, cl, policyKey, DefaultStaleEnactmentThreshold)
		Expect(err).ToNot(HaveOccurred())
		Expect(repaired).To(BeTrue(), "old-generation Progressing must not count as live holder")
	})

	It("is idempotent under repeated invocation", func() {
		policy := auditPolicy(2, 2, &oldUpdate)
		clb := buildClient()
		clb.WithRuntimeObjects(policy)
		clb.WithStatusSubresource(policy)
		cl := clb.Build()

		repaired, err := AuditUnavailableSlots(context.TODO(), cl, cl, policyKey, DefaultStaleEnactmentThreshold)
		Expect(err).ToNot(HaveOccurred())
		Expect(repaired).To(BeTrue())

		// Second run: count already 0 and timestamp is fresh -> grace defers.
		repaired, err = AuditUnavailableSlots(context.TODO(), cl, cl, policyKey, DefaultStaleEnactmentThreshold)
		Expect(err).ToNot(HaveOccurred())
		Expect(repaired).To(BeFalse())
	})

	It("returns zero-count no-op without status write", func() {
		policy := auditPolicy(2, 0, &oldUpdate)
		clb := buildClient()
		clb.WithRuntimeObjects(policy)
		clb.WithStatusSubresource(policy)
		cl := clb.Build()

		repaired, err := AuditUnavailableSlots(context.TODO(), cl, cl, policyKey, DefaultStaleEnactmentThreshold)
		Expect(err).ToNot(HaveOccurred())
		Expect(repaired).To(BeFalse())
	})
})

var _ = Describe("StaleEnactmentThreshold", func() {
	It("defaults to the derived threshold", func() {
		Expect(StaleEnactmentThreshold()).To(Equal(DefaultStaleEnactmentThreshold))
	})
	It("exceeds the worst-case apply cycle so a live applier is never freed", func() {
		Expect(DefaultStaleEnactmentThreshold).To(BeNumerically(">", worstCaseApplyCycle))
	})
	It("honors an env var that still exceeds the worst-case apply cycle", func() {
		override := worstCaseApplyCycle + 10*time.Minute
		GinkgoT().Setenv(StaleEnactmentThresholdEnvVar, override.String())
		Expect(StaleEnactmentThreshold()).To(Equal(override))
	})
	It("rejects an env var below the worst-case apply cycle, using the default", func() {
		GinkgoT().Setenv(StaleEnactmentThresholdEnvVar, "5m")
		Expect(StaleEnactmentThreshold()).To(Equal(DefaultStaleEnactmentThreshold))
	})
	It("falls back to default on unparsable value", func() {
		GinkgoT().Setenv(StaleEnactmentThresholdEnvVar, "bogus")
		Expect(StaleEnactmentThreshold()).To(Equal(DefaultStaleEnactmentThreshold))
	})
})
