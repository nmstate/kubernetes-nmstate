package nmstate_tests

import (
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	yaml "github.com/ghodss/yaml"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate.io/v1"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
)

var _ = Describe("Reporting State", func() {
	Context("periodically", func() {
		var (
			dsClient         appsv1client.DaemonSetInterface
			nodeNetworkState *nmstatev1.NodeNetworkState
			nodeName         string

			_ = BeforeEach(func() {

				By("Creating the daemon set to monitor state")
				manifest, err := ioutil.ReadFile(*manifests + "state-controller-ds.yaml")
				Expect(err).ShouldNot(HaveOccurred())

				var ds appsv1.DaemonSet
				err = yaml.Unmarshal(manifest, &ds)
				Expect(err).ShouldNot(HaveOccurred())

				dsClient = k8sClientset.AppsV1().DaemonSets(*nmstateNs)
				_, err = dsClient.Create(&ds)
				Expect(err).ShouldNot(HaveOccurred())

				By("Retrieving first node name")
				nodes, err := k8sClientset.CoreV1().Nodes().List(metav1.ListOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				nodeName = nodes.Items[0].ObjectMeta.Name

				By("Retrieving NodeNetworkState from node")
				nodeNetworkState, err = nmstateClientset.
					Nmstate().
					NodeNetworkStates(*nmstateNs).
					Get(nodeName, metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(nodeNetworkState.Spec.NodeName).To(Equal(nodeName))
			})
			_ = AfterEach(func() {
				dsClient.Delete("state-controller", &metav1.DeleteOptions{})
			})
		)

		It("should report correct node name", func() {
			Expect(nodeNetworkState.Spec.NodeName).To(Equal(nodeName))
		})

	})
})
