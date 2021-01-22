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
