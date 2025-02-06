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

var (
	AppliedFeaturesOpts = prometheus.GaugeOpts{
		Name: "kubernetes_nmstate_features_applied",
		Help: "Number of nmstate features applied labeled by its name",
	}

	AppliedFeatures = prometheus.NewGaugeVec(
		AppliedFeaturesOpts,
		[]string{"name"},
	)
	gaugeOpts = []prometheus.GaugeOpts{
		AppliedFeaturesOpts,
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
	return metricFamilies
}
