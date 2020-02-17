package nmstatectl

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"

	networkmanager "github.com/phoracek/networkmanager-go/src"
	"k8s.io/apimachinery/pkg/util/wait"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	walog = logf.Log.WithName("unavailable_link_workaround")
)

// There is a bug in Kernel/NetworkManager on systems with NetworkManager
// 1.20, where sometimes after disconnecting a NIC from a bonding, the NIC
// remains in 'unavailable' state and cannot be used for a new connection. This
// is likely caused by an issue with autonegotiation where the NIC appears to
// be disconnected and the only thing that can bring it available again is
// explicitly calling `ip link set <name> up` on it. In order to workaround
// this issue until it gets solved, we iterate all devices during `nmstatectl
// set` and if we find some with 'unavailable' we explicitly set them up.
func setUnavailableUp(stopCh chan struct{}) {
	nmClient, err := networkmanager.NewClientPrivate()
	if err != nil {
		walog.Error(err, "Failed to initialize NetworkManager client")
		return
	}
	defer nmClient.Close()

	wait.Until(func() {
		devices, err := nmClient.GetDevices()
		if err != nil {
			walog.Error(err, "Failed to list NetworkManager devices")
			return
		}

		for _, device := range devices {
			if device.Type == networkmanager.DeviceTypeEthernet && device.State == networkmanager.DeviceStateUnavailable {
				walog.Info("Ethernet interface in 'unavailable' state was found, setting explicitly UP", "iface", device.Interface)
				err := setLinkUp(device.Interface)
				if err != nil {
					walog.Error(err, "Failed to set interface UP", "iface", device.Interface)
				}
			}
		}
	}, time.Second, stopCh)
}

func setLinkUp(iface string) error {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command("ip", "link", "set", iface, "up")
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ip link set up failed, rc: %v, stdout: %v, stderr: %v", err, stdout.String(), stderr.String())
	}

	return nil
}
