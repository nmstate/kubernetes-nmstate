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

var (
	log            = logf.Log.WithName("backend")
	configBackend  Backend
)

// Backend defines the interface for network configuration backends.
type Backend interface {
	Show() (string, error)
	Set(desiredState nmstate.State, timeout time.Duration) (string, error)
	Commit() (string, error)
	Rollback() error
	Name() string
}

// InitBackend initializes the global backend based on the NMSTATE_BACKEND
// environment variable. Defaults to nmstate if not set.
func InitBackend() error {
	backendType := os.Getenv("NMSTATE_BACKEND")
	if backendType == "" {
		backendType = BackendNMState
	}

	log.Info("Initializing backend", "type", backendType)

	b, err := NewBackend(backendType)
	if err != nil {
		log.Error(err, "Failed to create backend, falling back to nmstate", "requestedType", backendType)
		configBackend = NewNMStateBackend()
		return nil
	}

	configBackend = b
	log.Info("Backend initialized successfully", "type", configBackend.Name())
	return nil
}

// GetBackend returns the current backend instance.
// If no backend has been initialized, it defaults to nmstate.
func GetBackend() Backend {
	if configBackend == nil {
		configBackend = NewNMStateBackend()
	}
	return configBackend
}

// Show is a convenience function that delegates to the current backend.
func Show() (string, error) {
	return GetBackend().Show()
}

// Name is a convenience function that returns the current backend name.
func Name() string {
	return GetBackend().Name()
}
