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

package operator

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/nmstate/kubernetes-nmstate/pkg/cluster"
	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/daemonset"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/deployment"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
)

type TestData struct {
	Ns                                                       string
	Nmstate                                                  nmstatev1.NMState
	WebhookKey, HandlerKey, CertManagerKey, ConsolePluginKey types.NamespacedName
	MetricsKey                                               *types.NamespacedName
	ManifestsDir                                             string
	ManifestFiles                                            []string
}

func NewOperatorTestData(ns string, manifestsDir string, manifestFiles []string) TestData {
	td := TestData{
		Ns: ns,
		Nmstate: nmstatev1.NMState{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nmstate",
				Namespace: ns,
			},
		},
		WebhookKey:       types.NamespacedName{Namespace: ns, Name: "nmstate-webhook"},
		HandlerKey:       types.NamespacedName{Namespace: ns, Name: "nmstate-handler"},
		CertManagerKey:   types.NamespacedName{Namespace: ns, Name: "nmstate-cert-manager"},
		ConsolePluginKey: types.NamespacedName{Namespace: ns, Name: "nmstate-console-plugin"},
		ManifestsDir:     manifestsDir,
		ManifestFiles:    manifestFiles,
	}
	// If there is a "servicemonitors" RBAC then nmstate-metrics deployment
	// should be  there
	for _, manifestFile := range manifestFiles {
		manifest, err := os.ReadFile(path.Join(manifestsDir, manifestFile))
		Expect(err).ToNot(HaveOccurred(), "should successfully open manifests to check if nmstate-metrics is needed")
		if strings.Contains(string(manifest), "servicemonitors") {
			td.MetricsKey = &types.NamespacedName{Namespace: ns, Name: "nmstate-metrics"}
			break
		}
	}
	return td
}

func InstallNMState(nmstate nmstatev1.NMState) {
	By(fmt.Sprintf("Creating NMState CR '%s'", nmstate.Name))
	err := testenv.Client.Create(context.TODO(), &nmstate)
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "NMState CR created without error")
}

func UninstallNMState(nmstate nmstatev1.NMState) {
	By(fmt.Sprintf("Deleting NMState CR '%s'", nmstate.Name))
	err := testenv.Client.Delete(context.TODO(), &nmstate, &client.DeleteOptions{})
	Expect(err).To(SatisfyAny(Succeed(), WithTransform(apierrors.IsNotFound, BeTrue())), "NMState CR successfully removed")
	EventuallyIsNotFound(types.NamespacedName{Name: nmstate.Name}, &nmstate, "should delete NMState CR")
}

func EventuallyIsNotFound(key types.NamespacedName, obj client.Object, msg string) {
	By(fmt.Sprintf("Wait for %+v deletion", key))
	EventuallyWithOffset(1, func() error {
		err := testenv.Client.Get(context.TODO(), key, obj)
		return err
	}, 120*time.Second, 1*time.Second).Should(WithTransform(apierrors.IsNotFound, BeTrue()), msg)
}

func EventuallyIsFound(key types.NamespacedName, obj client.Object, msg string) {
	By(fmt.Sprintf("Wait for %+v creation", key))
	EventuallyWithOffset(1, func() error {
		return testenv.Client.Get(context.TODO(), key, obj)
	}, 120*time.Second, 1*time.Second).Should(Succeed(), msg)
}

func UninstallNMStateAndWaitForDeletion(testData TestData) {
	UninstallNMState(testData.Nmstate)
	EventuallyOperandIsNotFound(testData)
}

func EventuallyOperandIsReady(testData TestData) {
	EventuallyOperandIsFound(testData)
	By("Wait daemonset handler is ready")
	daemonset.GetEventually(testData.HandlerKey).Should(daemonset.BeReady(), "should start handler daemonset")
	By("Wait deployment webhook is ready")
	deployment.GetEventually(testData.WebhookKey).Should(deployment.BeReady(), "should start webhook deployment")
	if !IsOpenShift() {
		By("Wait deployment cert-manager is ready")
		deployment.GetEventually(testData.CertManagerKey).Should(deployment.BeReady(), "should start cert-manager deployment")
	} else {
		By("Wait deployment console-plugin is ready")
		deployment.GetEventually(testData.ConsolePluginKey).Should(deployment.BeReady(), "should start console-plugin deployment")
	}
	if testData.MetricsKey != nil {
		By("Wait deployment metrics is ready")
		deployment.GetEventually(*testData.MetricsKey).Should(deployment.BeReady(), "should start metrics deployment")
	}
}

func EventuallyOperandIsNotFound(testData TestData) {
	EventuallyIsNotFound(testData.HandlerKey, &appsv1.DaemonSet{}, "should delete handler daemonset")
	EventuallyIsNotFound(testData.WebhookKey, &appsv1.Deployment{}, "should delete webhook deployment")
	if !IsOpenShift() {
		EventuallyIsNotFound(testData.CertManagerKey, &appsv1.Deployment{}, "should delete cert-manager deployment")
	} else {
		EventuallyIsNotFound(testData.ConsolePluginKey, &appsv1.Deployment{}, "should delete console-plugin deployment")
	}
	if testData.MetricsKey != nil {
		EventuallyIsNotFound(*testData.MetricsKey, &appsv1.Deployment{}, "should delete metrics deployment")
	}
	By("Wait for operand pods to terminate")
	Eventually(func() ([]corev1.Pod, error) {
		podList := corev1.PodList{}
		err := testenv.Client.List(
			context.TODO(),
			&podList,
			&client.ListOptions{Namespace: testData.Ns, LabelSelector: labels.Set{"app": "kubernetes-nmstate"}.AsSelector()},
		)
		return podList.Items, err
	}, 120*time.Second, time.Second).Should(BeEmpty(), "should terminate all the pods")
}

func EventuallyOperandIsFound(testData TestData) {
	EventuallyIsFound(testData.HandlerKey, &appsv1.DaemonSet{}, "should create handler daemonset")
	EventuallyIsFound(testData.WebhookKey, &appsv1.Deployment{}, "should create webhook deployment")
	if !IsOpenShift() {
		EventuallyIsFound(testData.CertManagerKey, &appsv1.Deployment{}, "should create cert-manager deployment")
	}
	if testData.MetricsKey != nil {
		EventuallyIsFound(*testData.MetricsKey, &appsv1.Deployment{}, "should create metrics deployment")
	}
}

func InstallOperator(operator TestData) {
	By(fmt.Sprintf("Creating NMState operator with namespace '%s'", operator.Ns))
	_, err := cmd.Run(
		"make",
		false,
		fmt.Sprintf("OPERATOR_NAMESPACE=%s", operator.Ns),
		fmt.Sprintf("HANDLER_NAMESPACE=%s", operator.Ns),
		"manifests",
	)
	Expect(err).ToNot(HaveOccurred())

	for _, manifest := range operator.ManifestFiles {
		_, err = cmd.Kubectl("apply", "-f", operator.ManifestsDir+manifest)
		Expect(err).ToNot(HaveOccurred())
	}

	deployment.GetEventually(types.NamespacedName{Namespace: operator.Ns, Name: "nmstate-operator"}).Should(deployment.BeReady())
}

func UninstallOperator(operator TestData) {
	By(fmt.Sprintf("Deleting namespace '%s'", operator.Ns))
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: operator.Ns,
		},
	}
	Expect(testenv.Client.Delete(context.TODO(), &ns)).To(SatisfyAny(Succeed(), WithTransform(apierrors.IsNotFound, BeTrue())))
	EventuallyIsNotFound(types.NamespacedName{Name: operator.Ns}, &ns, "should delete the namespace")
}

func IsOpenShift() bool {
	GinkgoHelper()
	isOpenShift, err := cluster.IsOpenShift(testenv.Client)
	Expect(err).ToNot(HaveOccurred())
	return isOpenShift
}
