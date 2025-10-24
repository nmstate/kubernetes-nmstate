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

package environment

import (
	"os"
	"time"

	"github.com/pkg/errors"
)

const (
	// DefaultPeriodicReconciliationInterval is the default interval for periodic reconciliation
	DefaultPeriodicReconciliationInterval = 30 * time.Minute
	// DefaultPeriodicWebhookCheckTimeout is the default timeout for webhook readiness checks during periodic reconciliation
	DefaultPeriodicWebhookCheckTimeout = 30 * time.Second
)

// IsOperator returns true when RUN_OPERATOR env var is present
func IsOperator() bool {
	_, runOperator := os.LookupEnv("RUN_OPERATOR")
	return runOperator
}

// IsWebhook returns true when RUN_WEBHOOK_SERVER env var is present
func IsWebhook() bool {
	_, runWebhook := os.LookupEnv("RUN_WEBHOOK_SERVER")
	return runWebhook
}

// IsCertManager return true when RUN_CERT_MANAGER env var is present
func IsCertManager() bool {
	_, runCertManager := os.LookupEnv("RUN_CERT_MANAGER")
	return runCertManager
}

// IsCertManager return true when RUN_CERT_MANAGER env var is present
func IsMetricsManager() bool {
	_, runMetricsManager := os.LookupEnv("RUN_METRICS_MANAGER")
	return runMetricsManager
}

// IsHandler returns true if it's not the operator or webhook server
func IsHandler() bool {
	return !IsWebhook() && !IsOperator() && !IsCertManager() && !IsMetricsManager()
}

// Returns node name runnig the pod
func NodeName() string {
	return os.Getenv("NODE_NAME")
}

func LookupAsDuration(varName string) (time.Duration, error) {
	duration := time.Duration(0)
	varValue, ok := os.LookupEnv(varName)
	if !ok {
		return duration, errors.Errorf("Failed to load %s from environment", varName)
	}

	duration, err := time.ParseDuration(varValue)
	if err != nil {
		return duration, errors.Wrapf(err, "Failed to convert %s value to time.Duration", varName)
	}
	return duration, nil
}

func GetEnvVar(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// PeriodicReconciliationInterval returns the periodic reconciliation interval from environment.
// Returns 0 if periodic reconciliation is disabled (value is "0").
// Returns DefaultPeriodicReconciliationInterval if not set or parse fails.
func PeriodicReconciliationInterval() time.Duration {
	interval := GetEnvVar("PERIODIC_RECONCILIATION_INTERVAL", DefaultPeriodicReconciliationInterval.String())
	if interval == "0" {
		return time.Duration(0) // disabled
	}
	duration, err := time.ParseDuration(interval)
	if err != nil {
		// Return default if parsing fails
		return DefaultPeriodicReconciliationInterval
	}
	return duration
}

// PeriodicWebhookCheckTimeout returns the timeout for webhook readiness checks during periodic reconciliation.
// Returns DefaultPeriodicWebhookCheckTimeout if not set or parse fails.
func PeriodicWebhookCheckTimeout() time.Duration {
	timeout := GetEnvVar("PERIODIC_WEBHOOK_CHECK_TIMEOUT", DefaultPeriodicWebhookCheckTimeout.String())
	duration, err := time.ParseDuration(timeout)
	if err != nil {
		// Return default if parsing fails
		return DefaultPeriodicWebhookCheckTimeout
	}
	return duration
}
