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

package env

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/test/environment"
)

var (
	cfg                 *rest.Config
	Client              client.Client         // You'll be using this client in your tests.
	KubeClient          *kubernetes.Clientset // You'll be using this client in your tests.
	testEnv             *envtest.Environment
	OperatorNamespace   string
	MonitoringNamespace string
)

func TestMain() {
	OperatorNamespace = environment.GetVarWithDefault("OPERATOR_NAMESPACE", "nmstate")
	MonitoringNamespace = environment.GetVarWithDefault("MONITORING_NAMESPACE", "monitoring")
}

func Start() {
	useExistingCluster := true
	testEnv = &envtest.Environment{
		UseExistingCluster: &useExistingCluster,
	}

	var err error
	cfg, err = testEnv.Start()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, cfg).ToNot(BeNil())

	err = nmstatev1.AddToScheme(scheme.Scheme)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	err = nmstatev1beta1.AddToScheme(scheme.Scheme)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	Client, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, Client).ToNot(BeNil())

	KubeClient, err = kubernetes.NewForConfig(cfg)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}

// PatchHandlerRetryConfig sets lower NNCP retry values for faster E2E tests.
// It patches the operator Deployment's env vars so that the operator re-renders
// the handler DaemonSet template with the new values, then waits for the
// handler pods to come up with the updated configuration.
// If the operator Deployment does not exist, the patching is skipped.
func PatchHandlerRetryConfig() {
	ctx := context.Background()

	// Patch the operator Deployment so it passes the lower retry values
	// to the handler DaemonSet template on its next reconciliation.
	// Use RetryOnConflict to handle optimistic concurrency conflicts (HTTP 409).
	desiredEnvVars := map[string]string{
		"NNCP_MAX_RETRIES":             "2",
		"NNCP_MAX_BACKOFF_SECONDS":     "5",
		"NNCP_INITIAL_BACKOFF_SECONDS": "1",
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		operatorDeploy := &appsv1.Deployment{}
		if err := Client.Get(ctx, client.ObjectKey{
			Namespace: OperatorNamespace,
			Name:      "nmstate-operator",
		}, operatorDeploy); err != nil {
			return err
		}

		// Containers[0] is the nmstate-operator container, which is always first.
		found := map[string]bool{}
		for i := range operatorDeploy.Spec.Template.Spec.Containers[0].Env {
			env := &operatorDeploy.Spec.Template.Spec.Containers[0].Env[i]
			if val, ok := desiredEnvVars[env.Name]; ok {
				env.Value = val
				found[env.Name] = true
			}
		}
		for name, val := range desiredEnvVars {
			if !found[name] {
				operatorDeploy.Spec.Template.Spec.Containers[0].Env = append(
					operatorDeploy.Spec.Template.Spec.Containers[0].Env,
					corev1.EnvVar{Name: name, Value: val},
				)
			}
		}

		return Client.Update(ctx, operatorDeploy)
	})
	if apierrors.IsNotFound(err) {
		fmt.Println("nmstate-operator Deployment not found, skipping retry config patching")
		return
	}
	Expect(err).ToNot(HaveOccurred())

	// If the handler DaemonSet already exists (handler e2e tests), wait for
	// the operator to re-render it with the new values and for pods to restart.
	// If it doesn't exist yet (operator e2e tests), skip — the operator will
	// create it with the correct values when the NMState CR is reconciled.
	ds := &appsv1.DaemonSet{}
	err = Client.Get(ctx, client.ObjectKey{
		Namespace: OperatorNamespace,
		Name:      "nmstate-handler",
	}, ds)
	if apierrors.IsNotFound(err) {
		fmt.Println("nmstate-handler DaemonSet not yet created, skipping wait for handler pods")
		return
	}
	Expect(err).ToNot(HaveOccurred())

	Eventually(func(g Gomega) {
		freshDS := &appsv1.DaemonSet{}
		err := Client.Get(ctx, client.ObjectKey{
			Namespace: OperatorNamespace,
			Name:      "nmstate-handler",
		}, freshDS)
		g.Expect(err).ToNot(HaveOccurred())

		// Verify the DaemonSet template has the updated env var.
		// Containers[0] is the handler container, which is always first.
		hasCorrectRetries := false
		expectedRetries := desiredEnvVars["NNCP_MAX_RETRIES"]
		for _, env := range freshDS.Spec.Template.Spec.Containers[0].Env {
			if env.Name == "NNCP_MAX_RETRIES" && env.Value == expectedRetries {
				hasCorrectRetries = true
				break
			}
		}
		g.Expect(hasCorrectRetries).To(BeTrue(), "DaemonSet should have NNCP_MAX_RETRIES=%s", expectedRetries)

		// Verify all handler pods are ready with the new config
		podList := corev1.PodList{}
		filterHandlers := client.MatchingLabels{"component": "kubernetes-nmstate-handler"}
		err = Client.List(ctx, &podList, filterHandlers, client.InNamespace(OperatorNamespace))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(podList.Items).ToNot(BeEmpty())
		for _, pod := range podList.Items {
			podReady := false
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
					podReady = true
					break
				}
			}
			g.Expect(podReady).To(BeTrue(), "Pod %s should be ready", pod.Name)

			// Containers[0] is the handler container, which is always first.
			hasRetries := false
			for _, env := range pod.Spec.Containers[0].Env {
				if env.Name == "NNCP_MAX_RETRIES" && env.Value == expectedRetries {
					hasRetries = true
					break
				}
			}
			g.Expect(hasRetries).To(BeTrue(), "Pod %s should have NNCP_MAX_RETRIES=%s", pod.Name, expectedRetries)
		}
	}, 5*time.Minute, 5*time.Second).Should(Succeed())
}
