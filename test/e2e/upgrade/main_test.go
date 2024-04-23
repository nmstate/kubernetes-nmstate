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

package upgrade

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	ginkgotypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/operator"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	knmstatereporter "github.com/nmstate/kubernetes-nmstate/test/reporter"
)

const (
	latestManifestsDir          = "build/_output/manifests/"
	previousReleaseManifestsDir = "test/e2e/upgrade/manifests/"
	ReadTimeout                 = 180 * time.Second
	ReadInterval                = 1 * time.Second
)

var (
	nodes            []string
	knmstateReporter *knmstatereporter.KubernetesNMStateReporter
)

var (
	manifestFiles = []string{
		"namespace.yaml",
		"service_account.yaml",
		"operator.yaml",
		"role.yaml",
		"role_binding.yaml",
	}
	latestOperator, previousReleaseOperator operator.TestData
)

func TestE2E(t *testing.T) {
	testenv.TestMain()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Upgrade E2E Test Suite")
}

var _ = BeforeSuite(func() {
	// Change to root directory some test expect that
	os.Chdir("../../../")

	latestOperator = operator.NewOperatorTestData("nmstate", latestManifestsDir, manifestFiles)
	previousReleaseOperator = operator.NewOperatorTestData("nmstate", previousReleaseManifestsDir, manifestFiles)

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	testenv.Start()

	knmstateReporter = knmstatereporter.New("test_logs/e2e/handler", testenv.OperatorNamespace, nodes)
	knmstateReporter.Cleanup()
})

var _ = ReportBeforeEach(func(specReport ginkgotypes.SpecReport) {
	knmstateReporter.ReportBeforeEach(specReport)
})

var _ = ReportAfterEach(func(specReport ginkgotypes.SpecReport) {
	knmstateReporter.ReportAfterEach(specReport)
})

func deletePolicy(name string) {
	By(fmt.Sprintf("Deleting policy %s", name))
	policy := &nmstatev1.NodeNetworkConfigurationPolicy{}
	policy.Name = name
	err := testenv.Client.Delete(context.TODO(), policy)
	if apierrors.IsNotFound(err) {
		return
	}
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	// Wait for policy to be removed
	EventuallyWithOffset(1, func() bool {
		err := testenv.Client.Get(context.TODO(), types.NamespacedName{Name: name}, &nmstatev1.NodeNetworkConfigurationPolicy{})
		return apierrors.IsNotFound(err)
	}, 60*time.Second, 1*time.Second).Should(BeTrue(), fmt.Sprintf("Policy %s not deleted", name))
}

func setDesiredStateWithPolicy(
	name string,
	desiredState shared.State,
) error {
	policy := nmstatev1.NodeNetworkConfigurationPolicy{}
	policy.Name = name
	key := types.NamespacedName{Name: name}
	err := testenv.Client.Get(context.TODO(), key, &policy)
	policy.Spec.DesiredState = desiredState
	if err != nil {
		if apierrors.IsNotFound(err) {
			return testenv.Client.Create(context.TODO(), &policy)
		}
		return err
	}
	err = testenv.Client.Update(context.TODO(), &policy)
	if err != nil {
		fmt.Println("Update error: " + err.Error())
	}
	return err
}

func setDesiredStateWithPolicyEventually(
	name string,
	desiredState shared.State,
) {
	Eventually(func() error {
		return setDesiredStateWithPolicy(name, desiredState)
	}, ReadTimeout, ReadInterval).ShouldNot(HaveOccurred(), fmt.Sprintf("Failed updating desired state : %s", desiredState))
}
