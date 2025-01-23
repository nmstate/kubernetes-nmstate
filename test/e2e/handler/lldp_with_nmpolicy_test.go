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

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("LLDP configuration with nmpolicy", func() {
	var lldpdPod *corev1.Pod

	lldpEnabledPolicyName := "lldp-enabled"
	lldpDisabledPolicyName := "lldp-disabled"

	configureLldpOnEthernetsCapture := func(enabled string) map[string]string {
		return map[string]string{
			"ethernets":      `interfaces.type=="ethernet"`,
			"ethernets-up":   `capture.ethernets | interfaces.state=="up"`,
			"ethernets-lldp": fmt.Sprintf(`capture.ethernets-up | interfaces.lldp.enabled:=%s`, enabled),
		}
	}

	interfacesWithLldpEnabledState := nmstate.NewState(`interfaces: "{{ capture.ethernets-lldp.interfaces }}"`)

	BeforeEach(func() {
		By("Starting lldpd at one node")
		lldpdPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "lldpd",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:    "lldpd",
					Image:   "quay.io/fedora/fedora-toolbox",
					Command: []string{"/bin/bash"},
					Args:    []string{"-c", "dnf install -y lldpd && lldpd -d"},
					SecurityContext: &corev1.SecurityContext{
						Privileged: ptr.To(true),
					},
				}},
				HostNetwork: true,
			},
		}
		Expect(testenv.Client.Create(context.Background(), lldpdPod)).To(Succeed())
		Eventually(func() (corev1.PodPhase, error) {
			if err := testenv.Client.Get(context.Background(), client.ObjectKeyFromObject(lldpdPod), lldpdPod); err != nil {
				return "", err
			}
			return lldpdPod.Status.Phase, nil
		}).WithTimeout(time.Minute).WithPolling(time.Second).Should(Equal(corev1.PodRunning))

		By("Enabling LLDP on up ethernet interfaces")
		setDesiredStateWithPolicyAndCapture(lldpEnabledPolicyName, interfacesWithLldpEnabledState, configureLldpOnEthernetsCapture("true"))
		policy.WaitForAvailablePolicy(lldpEnabledPolicyName)

		DeferCleanup(func() {
			deletePolicy(lldpEnabledPolicyName)

			By("Disabling LLDP on up ethernet interfaces")
			setDesiredStateWithPolicyAndCapture(lldpDisabledPolicyName, interfacesWithLldpEnabledState, configureLldpOnEthernetsCapture("false"))
			policy.WaitForAvailablePolicy(lldpDisabledPolicyName)
			deletePolicy(lldpDisabledPolicyName)

			By("Delete lldpd pod")
			Expect(testenv.Client.Delete(context.Background(), lldpdPod)).To(Succeed())
		})
	})

	It("should enable LLDP on all ethernet interfaces that are up and show neighbors", func() {
		Byf("Check %s has lldp enabled", primaryNic)
		for _, node := range nodes {
			Eventually(
				func() string {
					return lldpEnabled(node, primaryNic)
				},
				30*time.Second, 1*time.Second,
			).Should(Equal("true"), fmt.Sprintf("Interface %s should have enabled lldp", primaryNic))
		}

		Byf("Check %s has neighbors", primaryNic)
		Eventually(
			func() string {
				return lldpNeighbors(lldpdPod.Spec.NodeName, primaryNic)
			},
			5*time.Minute, time.Second,
		).ShouldNot(BeEmpty(), fmt.Sprintf("Interface %s should have lldp neighbors", primaryNic))
	})
})
