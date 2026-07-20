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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nmstatenode "github.com/nmstate/kubernetes-nmstate/pkg/node"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	"github.com/nmstate/kubernetes-nmstate/test/runner"
)

const (
	// vlanFilterTestThreshold is the threshold used during E2E tests.
	// We set it low so we can trigger stripping with a small number of VLANs.
	vlanFilterTestThreshold = "5"

	// Number of VLAN interfaces to create in the above-threshold test.
	// Must exceed vlanFilterTestThreshold.
	testVlanCount = 10

	// handlerRolloutTimeout is the time to wait for the handler DaemonSet
	// to complete a rollout after patching environment variables.
	handlerRolloutTimeout = 5 * time.Minute

	// handlerRolloutInterval is the polling interval for rollout checks.
	handlerRolloutInterval = 5 * time.Second

	// vlanFilterEnvVar is the name of the environment variable that
	// controls the VLAN filtering threshold.
	vlanFilterEnvVar = "VLAN_FILTER_INTERFACE_COUNT_THRESHOLD"
)

// createVlansOnNode creates VLAN sub-interfaces on the given node using
// `ip link add`. Returns the list of created VLAN interface names.
func createVlansOnNode(node, parentIface string, count int) []string {
	Byf("Creating %d VLAN interfaces on %s (parent: %s)", count, node, parentIface)

	// First bring up the parent interface
	_, err := runner.RunAtNode(node, "sudo", "ip", "link", "set", parentIface, "up")
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Failed to bring up parent interface %s on %s", parentIface, node)

	names := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		vlanID := fmt.Sprintf("%d", 3000+i) // Use high VLAN IDs to avoid conflicts
		vlanName := fmt.Sprintf("%s.%s", parentIface, vlanID)

		_, err := runner.RunAtNode(node,
			"sudo", "ip", "link", "add",
			"link", parentIface,
			"name", vlanName,
			"type", "vlan",
			"id", vlanID,
		)
		ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Failed to create VLAN %s on %s", vlanName, node)

		_, err = runner.RunAtNode(node, "sudo", "ip", "link", "set", vlanName, "up")
		ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Failed to bring up VLAN %s on %s", vlanName, node)

		names = append(names, vlanName)
	}

	return names
}

// deleteVlansOnNode removes VLAN sub-interfaces from the given node.
func deleteVlansOnNode(node, parentIface string, count int) {
	Byf("Deleting %d VLAN interfaces on %s (parent: %s)", count, node, parentIface)
	for i := 1; i <= count; i++ {
		vlanID := fmt.Sprintf("%d", 3000+i)
		vlanName := fmt.Sprintf("%s.%s", parentIface, vlanID)
		// Ignore errors — interface may already be gone
		runner.RunAtNode(node, "sudo", "ip", "link", "delete", vlanName) //nolint:errcheck
	}
}

// getHandlerDaemonSet returns the nmstate-handler DaemonSet.
func getHandlerDaemonSet() *appsv1.DaemonSet {
	ds := &appsv1.DaemonSet{}
	key := types.NamespacedName{
		Name:      "nmstate-handler",
		Namespace: "nmstate",
	}
	ExpectWithOffset(1, testenv.Client.Get(context.TODO(), key, ds)).To(Succeed())
	return ds
}

// setHandlerEnvVar patches the nmstate-handler DaemonSet to add/update
// an environment variable on the handler container. Returns the original
// env vars for restoration.
func setHandlerEnvVar(envName, envValue string) []corev1.EnvVar {
	ds := getHandlerDaemonSet()

	// Find the handler container
	var containerIdx int
	var found bool
	for i, c := range ds.Spec.Template.Spec.Containers {
		if c.Name == "nmstate-handler" || strings.Contains(c.Name, "handler") {
			containerIdx = i
			found = true
			break
		}
	}
	// If no container named "handler", use the first one
	if !found {
		containerIdx = 0
	}

	// Save original env vars
	origEnv := make([]corev1.EnvVar, len(ds.Spec.Template.Spec.Containers[containerIdx].Env))
	copy(origEnv, ds.Spec.Template.Spec.Containers[containerIdx].Env)

	// Check if the env var already exists
	envFound := false
	for j, e := range ds.Spec.Template.Spec.Containers[containerIdx].Env {
		if e.Name == envName {
			ds.Spec.Template.Spec.Containers[containerIdx].Env[j].Value = envValue
			envFound = true
			break
		}
	}
	if !envFound {
		ds.Spec.Template.Spec.Containers[containerIdx].Env = append(
			ds.Spec.Template.Spec.Containers[containerIdx].Env,
			corev1.EnvVar{Name: envName, Value: envValue},
		)
	}

	Byf("Setting %s=%s on handler DaemonSet", envName, envValue)
	ExpectWithOffset(1, testenv.Client.Update(context.TODO(), ds)).To(Succeed())

	return origEnv
}

// restoreHandlerEnvVars restores the handler DaemonSet to its original env vars.
func restoreHandlerEnvVars(origEnv []corev1.EnvVar) {
	ds := getHandlerDaemonSet()

	var containerIdx int
	var found bool
	for i, c := range ds.Spec.Template.Spec.Containers {
		if c.Name == "nmstate-handler" || strings.Contains(c.Name, "handler") {
			containerIdx = i
			found = true
			break
		}
	}
	if !found {
		containerIdx = 0
	}

	ds.Spec.Template.Spec.Containers[containerIdx].Env = origEnv

	By("Restoring handler DaemonSet environment variables")
	ExpectWithOffset(1, testenv.Client.Update(context.TODO(), ds)).To(Succeed())
}

// waitForHandlerRollout waits until all handler pods are ready and up-to-date.
func waitForHandlerRollout() {
	By("Waiting for handler DaemonSet rollout to complete")
	Eventually(func() bool {
		ds := getHandlerDaemonSet()
		return ds.Status.DesiredNumberScheduled == ds.Status.NumberReady &&
			ds.Status.DesiredNumberScheduled == ds.Status.UpdatedNumberScheduled &&
			ds.Status.DesiredNumberScheduled > 0
	}, handlerRolloutTimeout, handlerRolloutInterval).Should(BeTrue(),
		"Handler DaemonSet rollout did not complete in time")

	// Additional wait for pods to be fully ready
	Eventually(func() bool {
		podList := corev1.PodList{}
		err := testenv.Client.List(context.TODO(), &podList,
			client.InNamespace("nmstate"),
			client.MatchingLabels{"component": "kubernetes-nmstate-handler"},
		)
		if err != nil {
			return false
		}
		for _, pod := range podList.Items {
			if pod.Status.Phase != corev1.PodRunning {
				return false
			}
			for _, cs := range pod.Status.ContainerStatuses {
				if !cs.Ready {
					return false
				}
			}
		}
		return true
	}, handlerRolloutTimeout, handlerRolloutInterval).Should(BeTrue(),
		"Not all handler pods are running and ready")
}

var _ = Describe("[nns] NNS VLAN field filtering", func() {
	var (
		testNode    string
		vlanNames   []string
		parentIface string
	)

	BeforeEach(func() {
		testNode = nodes[0]
		parentIface = firstSecondaryNic
	})

	Context("when a small number of VLANs are configured below the default threshold", func() {
		const smallVlanCount = 3

		BeforeEach(func() {
			vlanNames = createVlansOnNode(testNode, parentIface, smallVlanCount)
		})

		AfterEach(func() {
			deleteVlansOnNode(testNode, parentIface, smallVlanCount)
			// Wait for NNS to reflect deletion
			for _, name := range vlanNames {
				Eventually(func() []string {
					return interfacesNameForNode(testNode)
				}, 2*nmstatenode.NetworkStateRefresh, time.Second).ShouldNot(ContainElement(name))
			}
		})

		It("should show VLANs in NNS with all fields preserved including verbose ones", func() {
			By("Waiting for NNS to include all created VLANs")
			Eventually(func() []string {
				return interfacesNameForNode(testNode)
			}, 2*nmstatenode.NetworkStateRefresh, time.Second).Should(ContainElements(vlanNames))

			By("Verifying VLAN interfaces have all fields (below threshold)")
			stateJSON := currentStateJSON(testNode)
			for _, vlanName := range vlanNames {
				path := fmt.Sprintf("interfaces.#(name==\"%s\")", vlanName)
				vlanData := gjson.ParseBytes(stateJSON).Get(path)
				Expect(vlanData.Exists()).To(BeTrue(), "VLAN %s should exist in NNS", vlanName)

				// Essential fields must be present
				Expect(vlanData.Get("name").String()).To(Equal(vlanName))
				Expect(vlanData.Get("type").String()).To(Equal("vlan"))
				Expect(vlanData.Get("state").String()).To(Equal("up"))
				Expect(vlanData.Get("vlan.base-iface").String()).To(Equal(parentIface))

				// Below threshold — verbose fields should also be present
				Expect(vlanData.Get("mtu").Exists()).To(BeTrue(),
					"VLAN %s should have mtu field below threshold", vlanName)
				Expect(vlanData.Get("mac-address").Exists()).To(BeTrue(),
					"VLAN %s should have mac-address field below threshold", vlanName)
			}

			By("Verifying non-VLAN interfaces are unaffected")
			ethPath := fmt.Sprintf("interfaces.#(name==\"%s\")", primaryNic)
			ethData := gjson.ParseBytes(stateJSON).Get(ethPath)
			Expect(ethData.Exists()).To(BeTrue(), "%s should exist in NNS", primaryNic)
			Expect(ethData.Get("mtu").Exists()).To(BeTrue(),
				"%s should have mtu field", primaryNic)
			Expect(ethData.Get("mac-address").Exists()).To(BeTrue(),
				"%s should have mac-address field", primaryNic)
		})
	})

	Context("when the interface count exceeds the VLAN filtering threshold", func() {
		var origEnv []corev1.EnvVar

		BeforeEach(func() {
			By("Lowering the VLAN filter threshold on the handler DaemonSet")
			origEnv = setHandlerEnvVar(vlanFilterEnvVar, vlanFilterTestThreshold)
			waitForHandlerRollout()

			By("Creating VLAN interfaces to exceed the lowered threshold")
			vlanNames = createVlansOnNode(testNode, parentIface, testVlanCount)
		})

		AfterEach(func() {
			By("Deleting test VLAN interfaces")
			deleteVlansOnNode(testNode, parentIface, testVlanCount)

			By("Restoring original handler DaemonSet env vars")
			restoreHandlerEnvVars(origEnv)
			waitForHandlerRollout()

			// Wait for NNS to reflect deletion
			for _, name := range vlanNames {
				Eventually(func() []string {
					return interfacesNameForNode(testNode)
				}, 2*nmstatenode.NetworkStateRefresh, time.Second).ShouldNot(ContainElement(name))
			}
		})

		It("should strip verbose fields from VLAN interfaces in NNS", func() {
			By("Waiting for NNS to include all created VLANs")
			Eventually(func() []string {
				return interfacesNameForNode(testNode)
			}, 2*nmstatenode.NetworkStateRefresh, time.Second).Should(ContainElements(vlanNames))

			By("Waiting for NNS to reflect filtered VLAN state")
			// The handler needs to reconcile with the new threshold
			// and produce a filtered NNS
			Eventually(func() bool {
				stateJSON := currentStateJSON(testNode)
				vlanPath := fmt.Sprintf("interfaces.#(name==\"%s\")", vlanNames[0])
				vlanData := gjson.ParseBytes(stateJSON).Get(vlanPath)
				if !vlanData.Exists() {
					return false
				}
				// Check that verbose fields have been stripped
				return !vlanData.Get("mtu").Exists() && !vlanData.Get("mac-address").Exists()
			}, 2*nmstatenode.NetworkStateRefresh, time.Second).Should(BeTrue(),
				"VLAN verbose fields should be stripped when interface count exceeds threshold")

			By("Verifying all VLAN interfaces have essential fields preserved")
			stateJSON := currentStateJSON(testNode)
			for _, vlanName := range vlanNames {
				vlanPath := fmt.Sprintf("interfaces.#(name==\"%s\")", vlanName)
				vlanData := gjson.ParseBytes(stateJSON).Get(vlanPath)
				Expect(vlanData.Exists()).To(BeTrue(), "VLAN %s should exist in NNS", vlanName)

				// Essential fields must be preserved
				Expect(vlanData.Get("name").String()).To(Equal(vlanName),
					"VLAN %s should have name field", vlanName)
				Expect(vlanData.Get("type").String()).To(Equal("vlan"),
					"VLAN %s should have type=vlan", vlanName)
				Expect(vlanData.Get("state").String()).ToNot(BeEmpty(),
					"VLAN %s should have state field", vlanName)
				Expect(vlanData.Get("vlan").Exists()).To(BeTrue(),
					"VLAN %s should have vlan config preserved", vlanName)
				Expect(vlanData.Get("vlan.base-iface").String()).To(Equal(parentIface),
					"VLAN %s should have correct base-iface", vlanName)
			}

			By("Verifying verbose fields are stripped from all VLAN interfaces")
			for _, vlanName := range vlanNames {
				vlanPath := fmt.Sprintf("interfaces.#(name==\"%s\")", vlanName)
				vlanData := gjson.ParseBytes(stateJSON).Get(vlanPath)

				Expect(vlanData.Get("mtu").Exists()).To(BeFalse(),
					"VLAN %s should NOT have mtu field when above threshold", vlanName)
				Expect(vlanData.Get("mac-address").Exists()).To(BeFalse(),
					"VLAN %s should NOT have mac-address field when above threshold", vlanName)
				Expect(vlanData.Get("lldp").Exists()).To(BeFalse(),
					"VLAN %s should NOT have lldp field when above threshold", vlanName)
				Expect(vlanData.Get("ethtool").Exists()).To(BeFalse(),
					"VLAN %s should NOT have ethtool field when above threshold", vlanName)
			}
		})

		It("should preserve all fields on non-VLAN interfaces", func() {
			By("Waiting for NNS to include all created VLANs")
			Eventually(func() []string {
				return interfacesNameForNode(testNode)
			}, 2*nmstatenode.NetworkStateRefresh, time.Second).Should(ContainElements(vlanNames))

			By("Verifying non-VLAN interfaces retain all fields")
			stateJSON := currentStateJSON(testNode)

			// Check the primary NIC (ethernet)
			ethPath := fmt.Sprintf("interfaces.#(name==\"%s\")", primaryNic)
			ethData := gjson.ParseBytes(stateJSON).Get(ethPath)
			Expect(ethData.Exists()).To(BeTrue(), "%s should exist in NNS", primaryNic)
			Expect(ethData.Get("type").String()).To(Equal("ethernet"))
			Expect(ethData.Get("mtu").Exists()).To(BeTrue(),
				"%s should still have mtu field", primaryNic)
			Expect(ethData.Get("mac-address").Exists()).To(BeTrue(),
				"%s should still have mac-address field", primaryNic)

			// Check the parent interface of the VLANs (also ethernet)
			parentPath := fmt.Sprintf("interfaces.#(name==\"%s\")", parentIface)
			parentData := gjson.ParseBytes(stateJSON).Get(parentPath)
			Expect(parentData.Exists()).To(BeTrue(), "%s should exist in NNS", parentIface)
			Expect(parentData.Get("mtu").Exists()).To(BeTrue(),
				"%s should still have mtu field", parentIface)
			Expect(parentData.Get("mac-address").Exists()).To(BeTrue(),
				"%s should still have mac-address field", parentIface)
		})

		It("should produce a valid NNS object within etcd size limits", func() {
			By("Waiting for NNS to include all created VLANs")
			Eventually(func() []string {
				return interfacesNameForNode(testNode)
			}, 2*nmstatenode.NetworkStateRefresh, time.Second).Should(ContainElements(vlanNames))

			By("Verifying NNS object is valid and within size limits")
			key := types.NamespacedName{Name: testNode}
			nns := nodeNetworkState(key)
			Expect(nns.Status.CurrentState.Raw).ToNot(BeEmpty(),
				"NNS currentState should not be empty")

			// etcd limit is 1.5MB (1572864 bytes). Our NNS should be
			// well under this even with many VLANs.
			nnsSize := len(nns.Status.CurrentState.Raw)
			Byf("NNS currentState size: %d bytes", nnsSize)
			Expect(nnsSize).To(BeNumerically("<", 1500000),
				"NNS currentState should be under 1.5 MB etcd limit")

			By("Verifying NNS has a recent successful update timestamp")
			Expect(nns.Status.LastSuccessfulUpdateTime.Time).To(
				BeTemporally(">", time.Now().Add(-5*time.Minute)),
				"NNS should have been updated recently",
			)
		})
	})

	Context("when VLANs are created and deleted via NNCP", func() {
		const nncpVlanID = "3099"

		BeforeEach(func() {
			By("Creating a VLAN via NNCP")
			updateDesiredStateAndWait(ifaceUpWithVlanUp(firstSecondaryNic, nncpVlanID))
		})

		AfterEach(func() {
			By("Removing VLAN via NNCP")
			updateDesiredStateAndWait(vlanAbsent(firstSecondaryNic, nncpVlanID))
			resetDesiredStateForNodes()
		})

		It("should show the VLAN in NNS with correct fields and remove it on cleanup", func() {
			expectedVlanName := fmt.Sprintf("%s.%s", firstSecondaryNic, nncpVlanID)

			By("Verifying VLAN appears in NNS for all worker nodes")
			for _, node := range nodes {
				Eventually(func() []string {
					return interfacesNameForNode(node)
				}, 2*nmstatenode.NetworkStateRefresh, time.Second).Should(ContainElement(expectedVlanName))
			}

			By("Verifying VLAN has correct metadata in NNS")
			for _, node := range nodes {
				stateJSON := currentStateJSON(node)
				vlanPath := fmt.Sprintf("interfaces.#(name==\"%s\")", expectedVlanName)
				vlanData := gjson.ParseBytes(stateJSON).Get(vlanPath)
				Expect(vlanData.Exists()).To(BeTrue())
				Expect(vlanData.Get("type").String()).To(Equal("vlan"))
				Expect(vlanData.Get("state").String()).To(Equal("up"))
				Expect(vlanData.Get("vlan.id").Int()).To(Equal(int64(3099)))
				Expect(vlanData.Get("vlan.base-iface").String()).To(Equal(firstSecondaryNic))
			}
		})
	})
})
