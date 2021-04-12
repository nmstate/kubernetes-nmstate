package certificate

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// get wraps controller-runtime client `Get` to ensure that client cache
// is ready, sometimes after controller-runtime manager is ready the
// cache is still not ready, specially if you webhook or plain runnable
// is being used since it miss some controller bits.
func (m *Manager) get(key types.NamespacedName, value client.Object) error {
	return wait.PollImmediate(5*time.Second, 30*time.Second, func() (bool, error) {
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
