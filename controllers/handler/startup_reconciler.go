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

// StartupReconciler implements a Runnable that performs explicit reconciliation
// of all NNCPs when the handler starts up. This ensures all policies are
// reconciled in a deterministic order with clear observability.
type StartupReconciler struct {
	Client     client.Client
	Cache      cache.Cache
	Log        logr.Logger
	Controller *NodeNetworkConfigurationPolicyReconciler
}

// Start implements manager.Runnable. It waits for the cache to sync, then
// explicitly reconciles all NNCPs in alphabetical order.
func (r *StartupReconciler) Start(ctx context.Context) error {
	log := r.Log.WithName("StartupReconciler")
	log.Info("Starting NNCP startup reconciliation")

	// Wait for cache to sync before attempting to list policies
	log.Info("Waiting for cache to sync before startup reconciliation")
	if !r.Cache.WaitForCacheSync(ctx) {
		log.Error(nil, "Cache sync failed, aborting startup reconciliation")
		return nil
	}
	log.Info("Cache synced successfully")

	// Wait for webhook to be ready before reconciling NNCPs
	// This prevents NNCP status updates from failing due to webhook certificate issues
	webhookTimeoutStr := environment.GetEnvVar("WEBHOOK_READINESS_TIMEOUT", "300s")
	webhookTimeout, err := time.ParseDuration(webhookTimeoutStr)
	if err != nil {
		log.Info("Invalid WEBHOOK_READINESS_TIMEOUT, using default 300s", "error", err)
		webhookTimeout = 300 * time.Second
	}

	webhookConfig := readiness.NewCheckerConfig(webhookTimeout)
	log.Info("Waiting for webhook to be ready before starting reconciliation",
		"timeout", webhookTimeout.String())
	if !readiness.WaitForWebhookReady(ctx, r.Client, webhookConfig) {
		log.Info("Webhook not ready within timeout, proceeding anyway (fail-open for startup reconciliation)")
	} else {
		log.Info("Webhook is ready, proceeding with reconciliation")
	}

	// Record start time for metrics
	startTime := time.Now()

	// Reconcile all policies using shared helper
	reconciledCount, totalPolicies, err := reconcileAllPolicies(
		ctx,
		r.Client,
		r.Controller,
		log,
		"startup",
	)
	if err != nil {
		return err
	}

	duration := time.Since(startTime)
	log.Info("Startup reconciliation completed",
		"policiesReconciled", reconciledCount,
		"totalPolicies", totalPolicies,
		"durationSeconds", duration.Seconds())

	// Update metrics
	monitoring.HandlerStartupReconciliationTotal.Inc()
	monitoring.HandlerStartupPoliciesReconciled.Set(float64(reconciledCount))
	monitoring.HandlerStartupDuration.Observe(duration.Seconds())

	// Mark startup reconciliation as complete
	r.Controller.MarkStartupReconciliationComplete()

	return nil
}

// NeedLeaderElection implements LeaderElectionRunnable.
// Startup reconciliation should happen on every handler pod independently,
// not just on the leader.
func (r *StartupReconciler) NeedLeaderElection() bool {
	return false
}
