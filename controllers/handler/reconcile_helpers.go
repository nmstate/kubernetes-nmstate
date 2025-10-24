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

package controllers

import (
	"context"
	"sort"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

// reconcileAllPolicies is a shared helper function that lists, sorts, and reconciles
// all NNCPs in alphabetical order. It's used by both startup and periodic reconcilers
// to ensure consistent behavior.
func reconcileAllPolicies(
	ctx context.Context,
	cl client.Client,
	controller *NodeNetworkConfigurationPolicyReconciler,
	log logr.Logger,
	reconciliationType string,
) (reconciledCount, totalPolicies int, err error) {
	// List all NNCPs
	policyList := &nmstatev1.NodeNetworkConfigurationPolicyList{}
	if err := cl.List(ctx, policyList); err != nil {
		log.Error(err, "Failed to list NodeNetworkConfigurationPolicies",
			"reconciliationType", reconciliationType)
		return 0, 0, err
	}

	// Sort policies alphabetically for deterministic ordering
	sort.Slice(policyList.Items, func(i, j int) bool {
		return policyList.Items[i].Name < policyList.Items[j].Name
	})

	log.Info("Starting reconciliation of NNCPs",
		"reconciliationType", reconciliationType,
		"totalPolicies", len(policyList.Items))

	// Reconcile each policy
	reconciledCount = 0
	for i := range policyList.Items {
		policy := &policyList.Items[i]
		log.Info("Triggering reconciliation for NNCP",
			"reconciliationType", reconciliationType,
			"policy", policy.Name,
			"generation", policy.Generation)

		request := reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      policy.Name,
				Namespace: policy.Namespace,
			},
		}

		// Trigger reconciliation
		result, err := controller.Reconcile(ctx, request)
		if err != nil {
			log.Error(err, "Error during reconciliation of NNCP",
				"reconciliationType", reconciliationType,
				"policy", policy.Name)
			// Continue with other policies even if one fails
			continue
		}

		if result.Requeue {
			log.Info("NNCP reconciliation requested requeue",
				"reconciliationType", reconciliationType,
				"policy", policy.Name,
				"requeueAfter", result.RequeueAfter)
		}

		reconciledCount++
	}

	return reconciledCount, len(policyList.Items), nil
}
