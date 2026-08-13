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

package node

import (
	"context"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	nmstateclient "github.com/nmstate/kubernetes-nmstate/pkg/client"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	"github.com/nmstate/kubernetes-nmstate/pkg/probe"
)

// worstCaseApplyCycle bounds how long ApplyDesiredState can legitimately run
// on the success path. The Progressing heartbeat is stamped once at
// NotifyProgressing and is not refreshed again until the apply returns, so a
// live holder's heartbeat can be this old while it is still applying:
//
//	probe.Select (sequential pre-apply gw+dns probes) <= ProbesTotalTimeout
//	nmstatectl.Set                                     <= DesiredStateConfigurationTimeout
//	probe.Run    (sequential post-apply probes)        <= ProbesTotalTimeout
//
// Derived from the source-of-truth timeouts so it tracks any change to them.
const worstCaseApplyCycle = nmstateclient.DesiredStateConfigurationTimeout + 2*probe.ProbesTotalTimeout

const (
	// DefaultStaleEnactmentThreshold is how old a Progressing enactment's
	// heartbeat must be before the audit considers its holder dead. It must
	// exceed the worst-case apply cycle (worstCaseApplyCycle) with margin;
	// otherwise the audit could classify a node that is still legitimately
	// applying as stale, free its slot, and let more than maxUnavailable
	// nodes reconfigure concurrently. The extra margin absorbs the gap
	// between stamping the heartbeat and entering the apply, plus clock skew.
	DefaultStaleEnactmentThreshold = worstCaseApplyCycle + 5*time.Minute

	// StaleEnactmentThresholdEnvVar overrides DefaultStaleEnactmentThreshold
	// (time.ParseDuration format, e.g. "20m").
	StaleEnactmentThresholdEnvVar = "NMSTATE_ENACTMENT_STALE_THRESHOLD"

	// AuditGraceWindow: if LastUnavailableNodeCountUpdate is younger than
	// this, the audit defers. A legitimate incrementer's NotifyProgressing
	// write may still be in flight; ghost slots are by definition old.
	AuditGraceWindow = 30 * time.Second
)

// StaleEnactmentThreshold returns the configured staleness threshold.
func StaleEnactmentThreshold() time.Duration {
	raw := environment.GetEnvVar(StaleEnactmentThresholdEnvVar, "")
	if raw == "" {
		return DefaultStaleEnactmentThreshold
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return DefaultStaleEnactmentThreshold
	}
	return parsed
}

// AuditUnavailableSlots recomputes UnavailableNodeCountMap[currentGeneration]
// from live enactments and repairs it downward if it exceeds the number of
// live holders. A live holder is an enactment of the policy's current
// generation with Progressing=True and a heartbeat younger than
// staleThreshold. Returns true if a repair was written.
//
// The repair is set-to-truth (never a blind decrement), so concurrent audits
// from multiple nodes are idempotent. The count is never raised.
func AuditUnavailableSlots(
	ctx context.Context,
	statusWriter client.Client,
	apiReader client.Reader,
	policyKey types.NamespacedName,
	staleThreshold time.Duration,
) (bool, error) {
	repaired := false
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		repaired = false
		policy := &nmstatev1.NodeNetworkConfigurationPolicy{}
		if err := apiReader.Get(ctx, policyKey, policy); err != nil {
			return err
		}

		if policy.Status.LastUnavailableNodeCountUpdate != nil &&
			time.Since(policy.Status.LastUnavailableNodeCountUpdate.Time) < AuditGraceWindow {
			return nil
		}

		generationKey := strconv.FormatInt(policy.Generation, 10)
		stored := 0
		if policy.Status.UnavailableNodeCountMap != nil {
			stored = policy.Status.UnavailableNodeCountMap[generationKey]
		}
		if stored == 0 {
			return nil
		}

		live, err := countLiveHolders(ctx, apiReader, policy, staleThreshold)
		if err != nil {
			return err
		}
		if live >= stored {
			return nil
		}

		policy.Status.UnavailableNodeCountMap[generationKey] = live
		now := metav1.Now()
		policy.Status.LastUnavailableNodeCountUpdate = &now
		if err := statusWriter.Status().Update(ctx, policy); err != nil {
			return err
		}
		repaired = true
		return nil
	})
	return repaired, err
}

func countLiveHolders(
	ctx context.Context,
	apiReader client.Reader,
	policy *nmstatev1.NodeNetworkConfigurationPolicy,
	staleThreshold time.Duration,
) (int, error) {
	enactments := nmstatev1beta1.NodeNetworkConfigurationEnactmentList{}
	policyLabelFilter := client.MatchingLabels{shared.EnactmentPolicyLabel: policy.Name}
	if err := apiReader.List(ctx, &enactments, policyLabelFilter); err != nil {
		return 0, err
	}
	live := 0
	for i := range enactments.Items {
		enactment := &enactments.Items[i]
		if enactment.Status.PolicyGeneration != policy.Generation {
			continue
		}
		progressing := enactment.Status.Conditions.Find(
			shared.NodeNetworkConfigurationEnactmentConditionProgressing)
		if progressing == nil || progressing.Status != corev1.ConditionTrue {
			continue
		}
		if time.Since(progressing.LastHeartbeatTime.Time) >= staleThreshold {
			continue
		}
		live++
	}
	return live, nil
}
