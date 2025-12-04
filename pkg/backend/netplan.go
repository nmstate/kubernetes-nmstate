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

// Package backend provides network configuration backend implementations
// nolint:dupl
package backend

import (
	"time"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/netplanctl"
)

// NetplanBackend implements the Backend interface using netplan via D-Bus
// nolint:dupl // This is a simple wrapper, duplication with nmstate backend is expected
type NetplanBackend struct{}

func NewNetplanBackend() *NetplanBackend {
	return &NetplanBackend{}
}

func (b *NetplanBackend) Show() (string, error) {
	// Get current netplan configuration via D-Bus
	// Note: In production, you might want to convert netplan format to nmstate format
	return netplanctl.Show()
}

func (b *NetplanBackend) Set(desiredState nmstate.State, timeout time.Duration) (string, error) {
	// Use netplan D-Bus interface with Try method (similar to nmstatectl checkpoint)
	// The Try method will automatically rollback after timeout if not committed
	return netplanctl.Set(desiredState, timeout)
}

func (b *NetplanBackend) Commit() (string, error) {
	// Commit the pending configuration via D-Bus
	return netplanctl.Commit()
}

func (b *NetplanBackend) Rollback() error {
	// Cancel/rollback pending configuration via D-Bus
	return netplanctl.Rollback()
}

func (b *NetplanBackend) Name() string {
	return "netplan"
}
