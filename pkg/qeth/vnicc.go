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

package qeth

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
)

const (
	sysfsQethBase = "/sys/devices/qeth"
	sysfsNetBase  = "/sys/class/net"
	filePerm      = 0600
)

// VniccConfig holds the desired VNIC characteristics for a qeth device.
type VniccConfig struct {
	Flooding         *bool `json:"flooding,omitempty"         yaml:"flooding,omitempty"`
	McastFlooding    *bool `json:"mcast-flooding,omitempty"   yaml:"mcast-flooding,omitempty"`
	Learning         *bool `json:"learning,omitempty"         yaml:"learning,omitempty"`
	LearningTimeout  *int  `json:"learning-timeout,omitempty" yaml:"learning-timeout,omitempty"`
	RxBcast          *bool `json:"rx-bcast,omitempty"         yaml:"rx-bcast,omitempty"`
	TakeoverSetvmac  *bool `json:"takeover-setvmac,omitempty" yaml:"takeover-setvmac,omitempty"`
	TakeoverLearning *bool `json:"takeover-learning,omitempty" yaml:"takeover-learning,omitempty"`
	BridgeInvisible  *bool `json:"bridge-invisible,omitempty" yaml:"bridge-invisible,omitempty"`
}

// Manager handles reading and writing qeth VNIC characteristics via sysfs.
type Manager struct {
	log logr.Logger
}

// NewManager creates a new qeth VNIC characteristics Manager.
func NewManager(log logr.Logger) *Manager {
	return &Manager{log: log.WithName("qeth-vnicc")}
}

// Apply writes VniccConfig to sysfs for the given interface using real sysfs paths.
func (m *Manager) Apply(ifaceName string, cfg VniccConfig) error {
	return m.applyWithPaths(sysfsQethBase, sysfsNetBase, ifaceName, cfg)
}

// ApplyWithBasePath writes VniccConfig using a custom base path.
// Used in unit tests to point at a fake tmpdir sysfs.
func (m *Manager) ApplyWithBasePath(basePath, ifaceName string, cfg VniccConfig) error {
	qethBase := filepath.Join(basePath, "devices", "qeth")
	netBase := filepath.Join(basePath, "class", "net")
	return m.applyWithPaths(qethBase, netBase, ifaceName, cfg)
}

// Read returns the current VniccConfig from real sysfs for the given interface.
func (m *Manager) Read(ifaceName string) (*VniccConfig, error) {
	return m.readWithPaths(sysfsQethBase, sysfsNetBase, ifaceName)
}

// ReadWithBasePath reads VniccConfig from a custom base path (for unit tests).
func (m *Manager) ReadWithBasePath(basePath, ifaceName string) (*VniccConfig, error) {
	qethBase := filepath.Join(basePath, "devices", "qeth")
	netBase := filepath.Join(basePath, "class", "net")
	return m.readWithPaths(qethBase, netBase, ifaceName)
}

// IsQethInterface returns true if the interface is backed by a qeth device (real sysfs).
func (m *Manager) IsQethInterface(ifaceName string) bool {
	return m.isQethWithPaths(sysfsQethBase, sysfsNetBase, ifaceName)
}

// IsQethInterfaceWithBasePath checks qeth status using a custom base path (for unit tests).
func (m *Manager) IsQethInterfaceWithBasePath(basePath, ifaceName string) bool {
	qethBase := filepath.Join(basePath, "devices", "qeth")
	netBase := filepath.Join(basePath, "class", "net")
	return m.isQethWithPaths(qethBase, netBase, ifaceName)
}

func (m *Manager) applyWithPaths(qethBase, netBase, ifaceName string, cfg VniccConfig) error {
	busID, err := m.resolveQethBusID(netBase, ifaceName)
	if err != nil {
		return fmt.Errorf("failed to resolve qeth bus ID for interface %s: %w", ifaceName, err)
	}

	vniccPath := filepath.Join(qethBase, busID, "vnicc")
	if _, err := os.Stat(vniccPath); os.IsNotExist(err) {
		return fmt.Errorf("vnicc path does not exist for %s (%s): not a qeth device or not in layer2 mode", ifaceName, busID)
	}

	m.log.Info("Applying vnicc configuration", "interface", ifaceName, "busID", busID)

	// IBM doc: learning_timeout MUST be set before enabling learning
	if cfg.LearningTimeout != nil {
		if err := writeAttr(vniccPath, "learning_timeout", strconv.Itoa(*cfg.LearningTimeout)); err != nil {
			return fmt.Errorf("failed to set learning_timeout: %w", err)
		}
		m.log.Info("Set vnicc attribute", "interface", ifaceName, "attr", "learning_timeout", "value", *cfg.LearningTimeout)
	}

	// Apply all boolean attributes; learning goes last (after timeout is set)
	attrs := []struct {
		name    string
		sysName string
		val     *bool
	}{
		{"flooding", "flooding", cfg.Flooding},
		{"mcast-flooding", "mcast_flooding", cfg.McastFlooding},
		{"rx-bcast", "rx_bcast", cfg.RxBcast},
		{"takeover-setvmac", "takeover_setvmac", cfg.TakeoverSetvmac},
		{"takeover-learning", "takeover_learning", cfg.TakeoverLearning},
		{"bridge-invisible", "bridge_invisible", cfg.BridgeInvisible},
		{"learning", "learning", cfg.Learning},
	}

	for _, a := range attrs {
		if a.val == nil {
			continue
		}
		val := boolToSysfs(*a.val)
		if err := writeAttr(vniccPath, a.sysName, val); err != nil {
			return fmt.Errorf("failed to set vnicc/%s=%s for %s: %w", a.sysName, val, ifaceName, err)
		}
		m.log.Info("Set vnicc attribute", "interface", ifaceName, "attr", a.sysName, "value", val)
	}

	return nil
}

func (m *Manager) readWithPaths(qethBase, netBase, ifaceName string) (*VniccConfig, error) {
	busID, err := m.resolveQethBusID(netBase, ifaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve qeth bus ID for %s: %w", ifaceName, err)
	}

	vniccPath := filepath.Join(qethBase, busID, "vnicc")
	if _, err := os.Stat(vniccPath); os.IsNotExist(err) {
		return nil, nil // not a qeth device — silently skip
	}

	cfg := &VniccConfig{}

	boolAttrs := []struct {
		sysName string
		target  **bool
	}{
		{"flooding", &cfg.Flooding},
		{"mcast_flooding", &cfg.McastFlooding},
		{"learning", &cfg.Learning},
		{"rx_bcast", &cfg.RxBcast},
		{"takeover_setvmac", &cfg.TakeoverSetvmac},
		{"takeover_learning", &cfg.TakeoverLearning},
		{"bridge_invisible", &cfg.BridgeInvisible},
	}

	for _, a := range boolAttrs {
		raw, err := readAttr(vniccPath, a.sysName)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading vnicc/%s: %w", a.sysName, err)
		}
		v := raw == "1"
		*a.target = &v
	}

	rawTimeout, err := readAttr(vniccPath, "learning_timeout")
	if err == nil {
		v, err := strconv.Atoi(rawTimeout)
		if err == nil {
			cfg.LearningTimeout = &v
		}
	}

	return cfg, nil
}

func (m *Manager) isQethWithPaths(qethBase, netBase, ifaceName string) bool {
	busID, err := m.resolveQethBusID(netBase, ifaceName)
	if err != nil {
		return false
	}
	vniccPath := filepath.Join(qethBase, busID, "vnicc")
	_, err = os.Stat(vniccPath)
	return err == nil
}

func (m *Manager) resolveQethBusID(netBase, ifaceName string) (string, error) {
	devicePath := filepath.Join(netBase, ifaceName, "device")
	resolved, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return "", fmt.Errorf("cannot resolve device symlink for %s: %w", ifaceName, err)
	}
	busID := filepath.Base(resolved)
	if busID == "" || busID == "." {
		return "", fmt.Errorf("could not determine bus ID from path %s", resolved)
	}
	return busID, nil
}

func writeAttr(vniccPath, attr, value string) error {
	return os.WriteFile(filepath.Join(vniccPath, attr), []byte(value), filePerm)
}

func readAttr(vniccPath, attr string) (string, error) {
	data, err := os.ReadFile(filepath.Join(vniccPath, attr))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func boolToSysfs(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
