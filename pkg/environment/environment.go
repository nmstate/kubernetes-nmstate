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
	"strconv"
	"time"

	"github.com/pkg/errors"
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
	return GetEnvVar("NODE_NAME", "")
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

// GetEnvVarAsInt reads a positive integer from an environment variable. It
// returns defaultVal when the variable is unset, and an error when the variable
// is set but is not a valid integer greater than zero.
func GetEnvVarAsInt(key string, defaultVal int) (int, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultVal, nil
	}
	intVal, err := strconv.Atoi(value)
	if err != nil || intVal <= 0 {
		return 0, errors.Errorf("%s must be a positive integer, got %q", key, value)
	}
	return intVal, nil
}

// GetEnvVarAsDuration reads a positive number of seconds from an environment
// variable and returns it as a time.Duration. It returns defaultVal when the
// variable is unset, and an error when the variable is set but is not a valid
// integer greater than zero.
func GetEnvVarAsDuration(key string, defaultVal time.Duration) (time.Duration, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultVal, nil
	}
	intVal, err := strconv.Atoi(value)
	if err != nil || intVal <= 0 {
		return 0, errors.Errorf("%s must be a positive integer number of seconds, got %q", key, value)
	}
	return time.Duration(intVal) * time.Second, nil
}
