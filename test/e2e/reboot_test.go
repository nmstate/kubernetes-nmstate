package e2e

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

var _ = Describe("Nmstate network connnections persistency", func() {
	Context("after rebooting a node with applied configuration", func() {
		var (
			macByNode = map[string]string{}
		)
		BeforeEach(func() {
			By("Apply policy")
			setDesiredStateWithPolicy("create-linux-bridge", linuxBrUp(bridge1))

			for _, node := range nodes {
				interfacesNameForNodeEventually(node).Should(ContainElement(bridge1))
				macByNode[node] = macAddress(node, bridge1)
				Expect(macByNode[node]).To(Equal(macAddress(node, *firstSecondaryNic)))
			}

			By("Move kubernetes configuration to prevent start up")
			_, errs := runAtNodes("sudo", "mv", "/etc/kubernetes", "/etc/kubernetes.bak")
			Expect(errs).ToNot(ContainElement(HaveOccurred()))

			By("Reboot the nodes")
			runAtNodes("sudo", "reboot")

			By("Wait for nodes to come back")
			_, errs = runAtNodes("true")
			Expect(errs).ToNot(ContainElement(HaveOccurred()))
		})

		AfterEach(func() {
			By("Move kubernetes configuration back")
			_, errs := runAtNodes("sudo", "mv", "/etc/kubernetes.bak", "/etc/kubernetes")
			Expect(errs).ToNot(ContainElement(HaveOccurred()))

			By("Wait for k8s to be ready")
			waitForNodesReady()

			By("Wait for Available NodeNetworkState")
			for _, node := range nodes {
				checkCondition(node, nmstatev1alpha1.NodeNetworkStateConditionAvailable).
					Should(Equal(corev1.ConditionTrue))
			}

			setDesiredStateWithPolicy("delete-linux-bridge", linuxBrAbsent(bridge1))
			for _, node := range nodes {
				interfacesNameForNodeEventually(node).ShouldNot(ContainElement(bridge1))
			}
			deletePolicy("create-linux-bridge")
			deletePolicy("delete-linux-bridge")
			resetDesiredStateForNodes()
		})
		It("should have nmstate connections before kubelet starts", func() {

			Consistently(func() []error {
				_, errs := runAtNodes("/usr/sbin/pidof", "kubelet")
				return errs
			}).ShouldNot(ContainElement(Succeed()), "Kubelet is not down")

			for _, node := range nodes {
				output, err := runAtNode(node, "sudo", "ip", "link", "show", bridge1)
				Expect(err).ToNot(HaveOccurred())
				obtainedLink := strings.ToLower(output)
				Expect(obtainedLink).To(ContainSubstring(macByNode[node]), "Linux bridge has unexpected mac address")
				Expect(obtainedLink).To(ContainSubstring(",up"), "Linux bridge is not ip")
			}
		})
	})
})
