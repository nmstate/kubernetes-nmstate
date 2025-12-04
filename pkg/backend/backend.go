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

package backend

import (
	"os"
	"time"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("backend")

// Backend defines the interface for network configuration backends
type Backend interface {
	// Show returns the current network state
	Show() (string, error)

	// Set applies the desired network state with a timeout
	// Returns output and error
	Set(desiredState nmstate.State, timeout time.Duration) (string, error)

	// Commit commits the pending network configuration changes
	Commit() (string, error)

	// Rollback rolls back pending network configuration changes
	Rollback() error

	// Name returns the backend name
	Name() string
}

// configBackend holds the network configuration backend instance
var configBackend Backend

// InitBackend initializes the network configuration backend based on environment variable
// If not set or invalid, defaults to nmstate backend
func InitBackend() error {
	backendType := os.Getenv("NMSTATE_BACKEND")
	if backendType == "" {
		backendType = BackendNMState
	}

	var err error
	configBackend, err = NewBackend(backendType)
	if err != nil {
		log.Error(err, "Failed to initialize backend, falling back to nmstate", "requestedBackend", backendType)
		configBackend = NewNMStateBackend()
	}

	log.Info("Initialized network configuration backend", "backend", configBackend.Name())
	return nil
}

// GetBackend returns the current backend instance
func GetBackend() Backend {
	if configBackend == nil {
		// Fallback to nmstate if not initialized
		configBackend = NewNMStateBackend()
	}
	return configBackend
}

// Show returns the current network state using the configured backend
func Show() (string, error) {
	return GetBackend().Show()
}

func Name() string {
	return GetBackend().Name()
}
