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

// IsHandler returns true if it's not the operator or webhook server
func IsHandler() bool {
	return !IsWebhook() && !IsOperator() && !IsCertManager()
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
