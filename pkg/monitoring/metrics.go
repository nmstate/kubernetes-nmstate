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

package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	pgo "github.com/prometheus/client_model/go"
	"k8s.io/utils/ptr"
)

const (
	// histogramBucketStart is the starting bucket size for histogram metrics (in seconds)
	histogramBucketStart = 0.1
	// histogramBucketFactor is the exponential growth factor for histogram buckets
	histogramBucketFactor = 2
	// histogramBucketCount is the number of buckets in histogram metrics
	histogramBucketCount = 10
)

var (
	AppliedFeaturesOpts = prometheus.GaugeOpts{
		Name: "kubernetes_nmstate_features_applied",
		Help: "Number of nmstate features applied labeled by its name",
	}

	AppliedFeatures = prometheus.NewGaugeVec(
		AppliedFeaturesOpts,
		[]string{"name"},
	)

	// Handler startup reconciliation metrics
	HandlerStartupReconciliationTotalOpts = prometheus.CounterOpts{
		Name: "kubernetes_nmstate_handler_startup_reconciliation_total",
		Help: "Total number of handler startup reconciliations performed",
	}

	HandlerStartupReconciliationTotal = prometheus.NewCounter(
		HandlerStartupReconciliationTotalOpts,
	)

	HandlerStartupPoliciesReconciledOpts = prometheus.GaugeOpts{
		Name: "kubernetes_nmstate_handler_startup_policies_reconciled",
		Help: "Number of policies reconciled during the last handler startup",
	}

	HandlerStartupPoliciesReconciled = prometheus.NewGauge(
		HandlerStartupPoliciesReconciledOpts,
	)

	HandlerStartupDurationOpts = prometheus.HistogramOpts{
		Name: "kubernetes_nmstate_handler_startup_duration_seconds",
		Help: "Duration of handler startup reconciliation in seconds",
		Buckets: prometheus.ExponentialBuckets(
			histogramBucketStart,
			histogramBucketFactor,
			histogramBucketCount,
		), // 0.1s to ~102s
	}

	HandlerStartupDuration = prometheus.NewHistogram(
		HandlerStartupDurationOpts,
	)

	// Handler periodic reconciliation metrics
	HandlerPeriodicReconciliationTotalOpts = prometheus.CounterOpts{
		Name: "kubernetes_nmstate_handler_periodic_reconciliation_total",
		Help: "Total number of periodic reconciliations performed",
	}

	HandlerPeriodicReconciliationTotal = prometheus.NewCounter(
		HandlerPeriodicReconciliationTotalOpts,
	)

	HandlerPeriodicPoliciesReconciledOpts = prometheus.GaugeOpts{
		Name: "kubernetes_nmstate_handler_periodic_policies_reconciled",
		Help: "Number of policies reconciled during the last periodic reconciliation",
	}

	HandlerPeriodicPoliciesReconciled = prometheus.NewGauge(
		HandlerPeriodicPoliciesReconciledOpts,
	)

	HandlerPeriodicDurationOpts = prometheus.HistogramOpts{
		Name: "kubernetes_nmstate_handler_periodic_duration_seconds",
		Help: "Duration of periodic reconciliation in seconds",
		Buckets: prometheus.ExponentialBuckets(
			histogramBucketStart,
			histogramBucketFactor,
			histogramBucketCount,
		), // 0.1s to ~102s
	}

	HandlerPeriodicDuration = prometheus.NewHistogram(
		HandlerPeriodicDurationOpts,
	)

	HandlerPeriodicIntervalOpts = prometheus.GaugeOpts{
		Name: "kubernetes_nmstate_handler_periodic_interval_seconds",
		Help: "Configured periodic reconciliation interval in seconds (0 if disabled)",
	}

	HandlerPeriodicInterval = prometheus.NewGauge(
		HandlerPeriodicIntervalOpts,
	)

	gaugeOpts = []prometheus.GaugeOpts{
		AppliedFeaturesOpts,
		HandlerStartupPoliciesReconciledOpts,
		HandlerPeriodicPoliciesReconciledOpts,
		HandlerPeriodicIntervalOpts,
	}
	counterOpts = []prometheus.CounterOpts{
		HandlerStartupReconciliationTotalOpts,
		HandlerPeriodicReconciliationTotalOpts,
	}
	histogramOpts = []prometheus.HistogramOpts{
		HandlerStartupDurationOpts,
		HandlerPeriodicDurationOpts,
	}
)

func Families() []pgo.MetricFamily {
	metricFamilies := []pgo.MetricFamily{}
	for _, gauge := range gaugeOpts {
		metricTypeGauge := pgo.MetricType_GAUGE
		metricFamilies = append(metricFamilies, pgo.MetricFamily{
			Name: ptr.To(gauge.Name),
			Help: ptr.To(gauge.Help),
			Type: &metricTypeGauge,
		})
	}
	for _, counter := range counterOpts {
		metricTypeCounter := pgo.MetricType_COUNTER
		metricFamilies = append(metricFamilies, pgo.MetricFamily{
			Name: ptr.To(counter.Name),
			Help: ptr.To(counter.Help),
			Type: &metricTypeCounter,
		})
	}
	for i := range histogramOpts {
		metricTypeHistogram := pgo.MetricType_HISTOGRAM
		metricFamilies = append(metricFamilies, pgo.MetricFamily{
			Name: ptr.To(histogramOpts[i].Name),
			Help: ptr.To(histogramOpts[i].Help),
			Type: &metricTypeHistogram,
		})
	}
	return metricFamilies
}
