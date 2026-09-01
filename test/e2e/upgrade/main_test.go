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
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	ginkgotypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/test/doc"
	"github.com/nmstate/kubernetes-nmstate/test/e2e/operator"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	knmstatereporter "github.com/nmstate/kubernetes-nmstate/test/reporter"
)

const (
	latestManifestsDir          = "build/_output/manifests/kubernetes-nmstate/templates/"
	previousReleaseManifestsDir = "test/e2e/upgrade/manifests/"
	ReadInterval                = 1 * time.Second
	// PolicyAvailableTimeout is the time given to a policy to become Available.
	PolicyAvailableTimeout = 5 * time.Minute
	// ReReconcileTimeout is the time given to all the policies to be re-reconciled
	// after the operator upgrade.
	ReReconcileTimeout = 5 * time.Minute
	// APIRetryTimeout is the time transient apiserver errors are retried for.
	APIRetryTimeout = 30 * time.Second
	// DeleteTimeout is the time given to a policy to be deleted.
	DeleteTimeout = 60 * time.Second
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

	By("Getting nmstate-enabled node list from cluster")
	podList := corev1.PodList{}
	filterHandlers := client.MatchingLabels{"component": "kubernetes-nmstate-handler"}
	Expect(testenv.Client.List(context.TODO(), &podList, filterHandlers)).To(Succeed())
	for _, pod := range podList.Items {
		nodes = append(nodes, pod.Spec.NodeName)
	}

	knmstateReporter = knmstatereporter.New("test_logs/e2e/handler", testenv.OperatorNamespace, nodes)
	knmstateReporter.Cleanup()
})

var _ = ReportBeforeEach(func(specReport ginkgotypes.SpecReport) {
	knmstateReporter.ReportBeforeEach(specReport)
})

var _ = ReportAfterEach(func(specReport ginkgotypes.SpecReport) {
	knmstateReporter.ReportAfterEach(specReport)
})

// terminalError wraps an error that must not be retried by retryUntil.
type terminalError struct{ cause error }

func (e *terminalError) Error() string { return e.cause.Error() }
func (e *terminalError) Unwrap() error { return e.cause }

// retryUntil runs fn until it succeeds or the timeout expires, returning the
// last observed error. It is used instead of gomega's Eventually where the
// caller needs to react to the failure instead of failing the spec. fn is
// always called at least once. A *terminalError returned by fn stops retrying
// immediately.
func retryUntil(timeout time.Duration, fn func() error) error {
	deadline := time.Now().Add(timeout)
	for {
		err := fn()
		if err == nil {
			return nil
		}
		var te *terminalError
		if errors.As(err, &te) {
			return te.cause
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(ReadInterval)
	}
}

// waitForPolicyAvailable waits until the policy becomes Available, failing
// immediately (without waiting for the timeout) when any NNCE has reached the
// terminal FailedToConfigure state, which means all handler retries are
// exhausted.  The NNCE message is included in the error so the actionable
// nmstatectl output is not hidden behind a generic timeout.
//
// Stale NNCP conditions are not a concern at the two call sites in this suite:
//   - initial apply: the NNCP is freshly created, so there are no pre-existing
//     conditions that could be mistaken for convergence.
//   - post-upgrade: the caller has already verified a post-upgrade heartbeat via
//     allPoliciesReReconciled, so the Available condition is guaranteed to
//     reflect the new operator's reconciliation pass.
func waitForPolicyAvailable(name string, timeout time.Duration) error {
	return retryUntil(timeout, func() error {
		// Check for terminal NNCE failures before reading the policy so that a
		// terminally-failed policy (which will never become Available) is reported
		// immediately with the actionable NNCE message rather than timing out.
		enactments := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
		if err := testenv.Client.List(
			context.TODO(),
			&enactments,
			client.MatchingLabels{shared.EnactmentPolicyLabel: name},
		); err != nil {
			return err // transient, retry
		}
		for i := range enactments.Items {
			nnce := &enactments.Items[i]
			failing := nnce.Status.Conditions.Find(
				shared.NodeNetworkConfigurationEnactmentConditionFailing,
			)
			if failing != nil &&
				failing.Status == corev1.ConditionTrue &&
				failing.Reason == shared.NodeNetworkConfigurationEnactmentConditionFailedToConfigure {
				return &terminalError{
					cause: fmt.Errorf(
						"enactment %s failed terminally (FailedToConfigure): %s",
						nnce.Name, failing.Message,
					),
				}
			}
		}

		policy := nmstatev1.NodeNetworkConfigurationPolicy{}
		if err := testenv.Client.Get(context.TODO(), types.NamespacedName{Name: name}, &policy); err != nil {
			return err // transient, retry
		}
		available := policy.Status.Conditions.Find(
			shared.NodeNetworkConfigurationPolicyConditionAvailable,
		)
		if available == nil || available.Status != corev1.ConditionTrue {
			return fmt.Errorf("policy %s is not Available yet", name)
		}
		return nil
	})
}

// policyExists checks if the policy is present at the apiserver retrying
// transient errors, a missing policy is not an error.
func policyExists(name string) (bool, error) {
	exists := false
	err := retryUntil(APIRetryTimeout, func() error {
		err := testenv.Client.Get(
			context.TODO(),
			types.NamespacedName{Name: name},
			&nmstatev1.NodeNetworkConfigurationPolicy{},
		)
		if apierrors.IsNotFound(err) {
			exists = false
			return nil
		}
		if err != nil {
			return err
		}
		exists = true
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("failed checking if policy %s exists: %w", name, err)
	}
	return exists, nil
}

func deletePolicy(name string) error {
	By(fmt.Sprintf("Deleting policy %s", name))
	err := retryUntil(APIRetryTimeout, func() error {
		policy := &nmstatev1.NodeNetworkConfigurationPolicy{}
		policy.Name = name
		err := testenv.Client.Delete(context.TODO(), policy)
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("failed deleting policy %s: %w", name, err)
	}

	// Wait for policy to be removed, transient read errors are retried until the
	// timeout expires so they don't end the wait prematurely.
	err = retryUntil(DeleteTimeout, func() error {
		err := testenv.Client.Get(
			context.TODO(),
			types.NamespacedName{Name: name},
			&nmstatev1.NodeNetworkConfigurationPolicy{},
		)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("policy %s is still present", name)
	})
	if err != nil {
		return fmt.Errorf("failed waiting for policy %s to be deleted: %w", name, err)
	}
	return nil
}

// policyEnactmentNames returns the names of the enactments the policy has at
// the apiserver, retrying transient errors.
func policyEnactmentNames(name string) ([]string, error) {
	names := []string{}
	err := retryUntil(APIRetryTimeout, func() error {
		enactments := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
		err := testenv.Client.List(
			context.TODO(),
			&enactments,
			client.MatchingLabels{shared.EnactmentPolicyLabel: name},
		)
		if err != nil {
			return err
		}
		names = names[:0]
		for i := range enactments.Items {
			names = append(names, enactments.Items[i].Name)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed listing the enactments of policy %s: %w", name, err)
	}
	return names, nil
}

// updatePolicyDesiredState replaces the desired state of an existing policy and
// returns the policy generation resulting from the update.
func updatePolicyDesiredState(name string, desiredState shared.State) (int64, error) {
	generation := int64(0)
	err := retryUntil(APIRetryTimeout, func() error {
		policy := nmstatev1.NodeNetworkConfigurationPolicy{}
		if err := testenv.Client.Get(context.TODO(), types.NamespacedName{Name: name}, &policy); err != nil {
			return err
		}
		policy.Spec.DesiredState = desiredState
		if err := testenv.Client.Update(context.TODO(), &policy); err != nil {
			return err
		}
		generation = policy.Generation
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed updating desired state of policy %s: %w", name, err)
	}
	return generation, nil
}

// waitForPolicyGenerationApplied waits until all the enactments of the policy
// have applied the given policy generation successfully. Checking the
// enactments' policyGeneration is mandatory since the policy conditions don't
// have an observedGeneration and can still reflect the previous desired state.
// The enactments observed before the update are expected to be all present, so
// a partial list cannot be mistaken for a converged one, and the enactments
// showing up afterwards have to converge too.
func waitForPolicyGenerationApplied(name string, generation int64, expectedEnactments []string, timeout time.Duration) error {
	return retryUntil(timeout, func() error {
		enactments := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
		err := testenv.Client.List(
			context.TODO(),
			&enactments,
			client.MatchingLabels{shared.EnactmentPolicyLabel: name},
		)
		if err != nil {
			return err
		}

		if len(expectedEnactments) == 0 && len(enactments.Items) == 0 {
			// Nothing can be observed at the enactments, so convergence cannot be
			// proven, callers have to skip the wait when no enactment is expected.
			return fmt.Errorf("policy %s has no enactment to verify", name)
		}

		converged := map[string]struct{}{}
		for i := range enactments.Items {
			enactment := &enactments.Items[i]
			if enactment.Status.PolicyGeneration != generation {
				return fmt.Errorf(
					"enactment %s has applied policy generation %d, expected %d",
					enactment.Name, enactment.Status.PolicyGeneration, generation,
				)
			}
			condition := enactment.Status.Conditions.Find(
				shared.NodeNetworkConfigurationEnactmentConditionAvailable,
			)
			if condition == nil || condition.Status != corev1.ConditionTrue {
				return fmt.Errorf("enactment %s is not Available", enactment.Name)
			}
			converged[enactment.Name] = struct{}{}
		}

		for _, expected := range expectedEnactments {
			if _, ok := converged[expected]; !ok {
				return fmt.Errorf("enactment %s of policy %s is missing", expected, name)
			}
		}
		return nil
	})
}

// cleanupDesiredState composes a single desired state removing everything the
// example has configured, so cleanup is done with one policy update instead of
// one update per interface, which could be coalesced by the handler workqueue.
func cleanupDesiredState(example doc.ExampleSpec) (shared.State, error) {
	if example.CleanupState == nil && len(example.IfaceNames) == 0 {
		return shared.State{}, nil
	}

	state := map[string]any{}
	if example.CleanupState != nil {
		if err := yaml.Unmarshal(example.CleanupState.Raw, &state); err != nil {
			return shared.State{}, fmt.Errorf("failed parsing cleanup state of example %s: %w", example.Name, err)
		}
	}

	interfaces := []any{}
	if rawInterfaces, ok := state["interfaces"]; ok {
		interfaces, ok = rawInterfaces.([]any)
		if !ok {
			return shared.State{}, fmt.Errorf("unexpected interfaces at cleanup state of example %s", example.Name)
		}
	}

	for _, ifaceName := range example.IfaceNames {
		if containsInterface(interfaces, ifaceName) {
			continue
		}
		interfaces = append(interfaces, map[string]any{
			"name":  ifaceName,
			"state": "absent",
		})
	}
	if len(interfaces) > 0 {
		state["interfaces"] = interfaces
	}

	rawState, err := yaml.Marshal(state)
	if err != nil {
		return shared.State{}, fmt.Errorf("failed composing cleanup state of example %s: %w", example.Name, err)
	}
	return shared.NewState(string(rawState)), nil
}

func containsInterface(interfaces []any, name string) bool {
	for _, rawIface := range interfaces {
		iface, ok := rawIface.(map[string]any)
		if !ok {
			continue
		}
		if iface["name"] == name {
			return true
		}
	}
	return false
}
