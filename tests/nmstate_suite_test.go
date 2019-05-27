/*
 * Copyright 2019 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nmstate_tests

import (
	"flag"
	"io/ioutil"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	yaml "github.com/ghodss/yaml"

	apiappsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"

	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/client/clientset/versioned"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/client/clientset/versioned/typed/nmstate.io/v1"
)

var (
	// Flags
	kubeconfig *string
	nmstateNs  *string
	manifests  *string

	// Scaffolding
	firstNodeName            string
	nmstatePodsClient        clientcorev1.PodInterface
	nmstateDaemonSetClient   clientappsv1.DaemonSetInterface
	defaultNNSs, nmstateNNSs nmstatev1.NodeNetworkStateInterface
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	reporters := make([]Reporter, 0)
	if ginkgo_reporters.JunitOutput != "" {
		reporters = append(reporters, ginkgo_reporters.NewJunitReporter())
	}
	RunSpecsWithDefaultAndCustomReporters(t, "kubernetes-nmstate suite", reporters)
}

var _ = BeforeSuite(func() {
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	Expect(err).ToNot(HaveOccurred())
	k8sClientset, err := kubernetes.NewForConfig(config)
	Expect(err).ToNot(HaveOccurred())
	nmstateClientset, err := nmstate.NewForConfig(config)
	Expect(err).ToNot(HaveOccurred())
	nmstatePodsClient = k8sClientset.CoreV1().Pods(*nmstateNs)
	nmstateDaemonSetClient = k8sClientset.AppsV1().DaemonSets(*nmstateNs)
	defaultNNSs = nmstateClientset.
		Nmstate().
		NodeNetworkStates("default")
	nmstateNNSs = nmstateClientset.
		Nmstate().
		NodeNetworkStates(*nmstateNs)

	By("Creating the daemon set to monitor state")
	manifest, err := ioutil.ReadFile(*manifests + "state-controller-ds.yaml")
	Expect(err).ToNot(HaveOccurred())

	var ds apiappsv1.DaemonSet
	err = yaml.Unmarshal(manifest, &ds)
	Expect(err).ToNot(HaveOccurred())

	_, err = nmstateDaemonSetClient.Create(&ds)
	Expect(err).ToNot(HaveOccurred())
	err = waitPodsReady()
	Expect(err).ToNot(HaveOccurred())

	By("Retrieving first node name")
	nodes, err := k8sClientset.CoreV1().Nodes().List(metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(nodes.Items).ToNot(BeEmpty())
	firstNodeName = nodes.Items[0].ObjectMeta.Name

})

var _ = AfterSuite(func() {

	By("Removing state-controller daemon set")
	nmstateDaemonSetClient.Delete("state-controller", &metav1.DeleteOptions{})
	err := waitPodsCleanup()
	Expect(err).ToNot(HaveOccurred())

	By("Removing node01 nodenetworkstates")
	defaultNNSs.Delete(firstNodeName, &metav1.DeleteOptions{})
	nmstateNNSs.Delete(firstNodeName, &metav1.DeleteOptions{})

})

var _ = AfterEach(func() {
	writePodsLogs(GinkgoWriter)
})

func init() {
	kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	nmstateNs = flag.String("namespace", "", "kubernetes-nmstate namespace")
	manifests = flag.String("manifests", "", "path to manifests to test")
	flag.Parse()
}
