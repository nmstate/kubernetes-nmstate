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
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	"github.com/nmstate/kubernetes-nmstate/pkg/monitoring"
	"github.com/nmstate/kubernetes-nmstate/pkg/webhook/readiness"
)

// PeriodicReconciler implements a Runnable that performs periodic reconciliation
// of all NNCPs at a configured interval. This ensures policies are continuously
// reconciled to maintain desired state even if events are missed.
type PeriodicReconciler struct {
	Client     client.Client
	Cache      cache.Cache
	Log        logr.Logger
	Controller *NodeNetworkConfigurationPolicyReconciler
	Interval   time.Duration
}

// Start implements manager.Runnable. It runs a ticker that periodically
// reconciles all NNCPs at the configured interval.
func (r *PeriodicReconciler) Start(ctx context.Context) error {
	log := r.Log.WithName("PeriodicReconciler")

	if r.Interval == 0 {
		log.Info("Periodic reconciliation is disabled (interval set to 0)")
		return nil
	}

	log.Info("Starting periodic NNCP reconciliation",
		"interval", r.Interval.String())

	// Wait for cache to sync before starting periodic reconciliation
	log.Info("Waiting for cache to sync before starting periodic reconciliation")
	if !r.Cache.WaitForCacheSync(ctx) {
		log.Error(nil, "Cache sync failed, aborting periodic reconciliation")
		return nil
	}
	log.Info("Cache synced successfully, starting periodic reconciliation ticker")

	// Update metric with configured interval
	monitoring.HandlerPeriodicInterval.Set(r.Interval.Seconds())

	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Periodic reconciliation stopped due to context cancellation")
			return nil
		case <-ticker.C:
			log.Info("Periodic reconciliation triggered")
			r.reconcileAll(ctx)
		}
	}
}

// reconcileAll reconciles all NNCPs in alphabetical order
func (r *PeriodicReconciler) reconcileAll(ctx context.Context) {
	log := r.Log.WithName("reconcileAll")
	startTime := time.Now()

	// Quick webhook readiness check before periodic reconciliation
	// Get timeout from environment (configurable via PERIODIC_WEBHOOK_CHECK_TIMEOUT)
	periodicWebhookCheckTimeout := environment.PeriodicWebhookCheckTimeout()
	webhookConfig := readiness.NewCheckerConfig(periodicWebhookCheckTimeout)
	webhookCtx, cancel := context.WithTimeout(ctx, periodicWebhookCheckTimeout)
	defer cancel()

	if !readiness.WaitForWebhookReady(webhookCtx, r.Client, webhookConfig) {
		log.Info("Webhook not ready, skipping this periodic reconciliation cycle")
		return
	}

	// Reconcile all policies using shared helper
	reconciledCount, totalPolicies, err := reconcileAllPolicies(
		ctx,
		r.Client,
		r.Controller,
		log,
		"periodic",
	)
	if err != nil {
		return
	}

	duration := time.Since(startTime)
	log.Info("Periodic reconciliation completed",
		"policiesReconciled", reconciledCount,
		"totalPolicies", totalPolicies,
		"durationSeconds", duration.Seconds(),
		"nextReconciliation", time.Now().Add(r.Interval).Format(time.RFC3339))

	// Update metrics
	monitoring.HandlerPeriodicReconciliationTotal.Inc()
	monitoring.HandlerPeriodicPoliciesReconciled.Set(float64(reconciledCount))
	monitoring.HandlerPeriodicDuration.Observe(duration.Seconds())
}

// NeedLeaderElection implements LeaderElectionRunnable.
// Periodic reconciliation should happen on every handler pod independently,
// not just on the leader.
func (r *PeriodicReconciler) NeedLeaderElection() bool {
	return false
}
