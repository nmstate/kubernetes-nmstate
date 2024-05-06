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
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/api/v1alpha1"
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
	err = nmstatev1alpha1.AddToScheme(scheme.Scheme)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	Client, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, Client).ToNot(BeNil())

	KubeClient, err = kubernetes.NewForConfig(cfg)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
}
