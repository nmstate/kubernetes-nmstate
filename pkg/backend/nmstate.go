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
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
)

// NMStateBackend implements the Backend interface using nmstatectl
// nolint:dupl // This is a simple wrapper, duplication with netplan backend is expected
type NMStateBackend struct{}

func NewNMStateBackend() *NMStateBackend {
	return &NMStateBackend{}
}

func (b *NMStateBackend) Show() (string, error) {
	return nmstatectl.Show()
}

func (b *NMStateBackend) Set(desiredState nmstate.State, timeout time.Duration) (string, error) {
	return nmstatectl.Set(desiredState, timeout)
}

func (b *NMStateBackend) Commit() (string, error) {
	return nmstatectl.Commit()
}

func (b *NMStateBackend) Rollback() error {
	return nmstatectl.Rollback()
}

func (b *NMStateBackend) Name() string {
	return "nmstate"
}
