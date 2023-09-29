/*
 * Copyright 2022 Kube Admission Webhook Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 *	  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package certificate

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pollInterval = 5 * time.Second
	pollTimeout  = 30 * time.Second
)

// get wraps controller-runtime client `Get` to ensure that client cache
// is ready, sometimes after controller-runtime manager is ready the
// cache is still not ready, specially if you webhook or plain runnable
// is being used since it miss some controller bits.
func (m *Manager) get(key types.NamespacedName, value client.Object) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (bool, error) {
		err := m.client.Get(context.TODO(), key, value)
		if err != nil {
			if _, cacheNotStarted := err.(*cache.ErrCacheNotStarted); cacheNotStarted {
				return false, nil
			} else {
				return true, err
			}
		}
		return true, nil
	})
}
