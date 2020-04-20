package certificate

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

func (m *Manager) get(key types.NamespacedName, value runtime.Object) error {
	return wait.PollImmediate(5*time.Second, 30*time.Second, func() (bool, error) {
		err := m.client.Get(context.TODO(), key, value)
		if err != nil {
			_, cacheNotStarted := err.(*cache.ErrCacheNotStarted)
			if cacheNotStarted {
				return false, nil
			} else {
				return true, err
			}
		}
		return true, nil
	})
}
