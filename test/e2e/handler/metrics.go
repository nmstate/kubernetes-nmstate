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

package handler

import (
	"strings"

	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	"github.com/nmstate/kubernetes-nmstate/test/runner"
)

func getMetrics(token string) map[string]string {
	bearer := "Authorization: Bearer " + token
	return indexMetrics(runner.RunAtMetricsPod("curl", "-s", "-k", "--header",
		bearer, "--header", "X-Authorization-Classification: notsecret", ":8089", "https://127.0.0.1:8443/metrics"))
}

func getPrometheusToken() (string, error) {
	const (
		prometheusPod = "prometheus-k8s-0"
		container     = "prometheus"
		tokenPath     = "/var/run/secrets/kubernetes.io/serviceaccount/token" // #nosec G101
	)

	return cmd.Kubectl("exec", "-n", testenv.MonitoringNamespace, prometheusPod, "-c", container, "--", "cat", tokenPath)
}

func indexMetrics(metrics string) map[string]string {
	metricsMap := map[string]string{}
	for _, metric := range strings.Split(metrics, "\n") {
		if strings.Contains(metric, "#") { // Ignore comments
			continue
		}
		metricSplit := strings.Split(metric, " ")
		if len(metricSplit) != 2 {
			continue
		}
		metricsMap[metricSplit[0]] = metricSplit[1]
	}
	return metricsMap
}
