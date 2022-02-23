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

package node

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	NetworkStateRefresh          = time.Minute
	NetworkStateRefreshMaxFactor = 0.1
)

// NodeNetworkStateRefreshWithJitter add some jitter to to the refresh rate so it does
// not hit apiserver at the same time.
func NetworkStateRefreshWithJitter() time.Duration {
	return wait.Jitter(NetworkStateRefresh, NetworkStateRefreshMaxFactor)
}
