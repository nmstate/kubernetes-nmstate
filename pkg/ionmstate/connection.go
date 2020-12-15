package ionmstate

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/varlink/go/varlink"

	"k8s.io/apimachinery/pkg/util/wait"
)

func NewConnection() (*varlink.Connection, error) {
	varlinkSo, ok := os.LookupEnv("NMSTATE_VARLINK_SOCKET")
	if !ok {
		return nil, fmt.Errorf("Failed to load varlink socket from environment %s", "NMSTATE_VARLINK_SOCKET")
	}
	var c *varlink.Connection
	var varlinkErr error
	err := wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		c, varlinkErr = varlink.NewConnection(context.Background(), varlinkSo)
		if varlinkErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed connecting to nmstate varlink server: %v, %w", err, varlinkErr)
	}
	return c, nil
}
