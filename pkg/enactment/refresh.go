package enactment

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	EnactmentRefresh          = 5 * time.Hour
	EnactmentRefreshMaxFactor = 0.1
)

// RefreshWithJitter adds jitter to the refresh rate so it does
// not hit apiserver at the same time.
func RefreshWithJitter() time.Duration {
	return wait.Jitter(EnactmentRefresh, EnactmentRefreshMaxFactor)
}
