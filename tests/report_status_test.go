package nmstate_tests

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	yaml "github.com/ghodss/yaml"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate.io/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
)

func IsReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady &&
			condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func WaitPodsCleanup() error {
	return wait.Poll(10, 30*time.Second, func() (bool, error) {
		pods, err := nmstatePodsClient.List(metav1.ListOptions{})
		if err != nil {
			return true, fmt.Errorf("error listing pods: %v", err)
		}
		return len(pods.Items) == 0, nil
	})
}

func WaitPodsReady() error {
	return wait.Poll(10, 30*time.Second, func() (bool, error) {
		pods, err := nmstatePodsClient.List(metav1.ListOptions{})
		if err != nil {
			return true, fmt.Errorf("error listing pods: %v", err)
		}
		if len(pods.Items) == 0 {
			return false, nil
		}
		for _, pod := range pods.Items {
			if !IsReady(pod) {
				return false, nil
			}
		}
		return true, nil
	})
}

func GetPodsLogs() (string, error) {
	podLogOpts := corev1.PodLogOptions{}
	pods, err := nmstatePodsClient.List(metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("error listing pods: %v", err)
	}
	buffer := new(bytes.Buffer)
	for _, pod := range pods.Items {
		req := nmstatePodsClient.GetLogs(pod.Name, &podLogOpts)
		podLogs, err := req.Stream()
		if err != nil {
			fmt.Printf("error in opening stream: %s\n", err)
			continue
		}
		defer podLogs.Close()
		_, err = io.Copy(buffer, podLogs)
		if err != nil {
			fmt.Printf("error in copy information from podLogs to buf: %s\n", err)
			continue
		}

	}
	return buffer.String(), nil
}

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
				err = WaitPodsReady()
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
				podsLogs, err := GetPodsLogs()
				Expect(err).ShouldNot(HaveOccurred())
				fmt.Println(podsLogs)
				dsClient.Delete("state-controller", &metav1.DeleteOptions{})
				err = WaitPodsCleanup()
				Expect(err).ShouldNot(HaveOccurred())
			})
		)

		It("should report correct node name", func() {
			Expect(nodeNetworkState.Spec.NodeName).To(Equal(nodeName))
		})

	})
})
