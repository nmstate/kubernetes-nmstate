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

package qeth_test

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nmstate/kubernetes-nmstate/pkg/qeth"
)

const (
	filePerm = 0600
)

func TestQeth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "qeth vnicc Suite")
}

// tHelper is the minimal interface satisfied by both *testing.T and GinkgoT().
// GinkgoT() returns FullGinkgoTInterface which does NOT implement *testing.T directly,
// so we define a common interface with only the methods we need.
type tHelper interface {
	TempDir() string
	Fatal(args ...any)
}

// fakeQethEnv sets up a fake sysfs structure for testing without real hardware.
func fakeQethEnv(t tHelper, busID, ifaceName string, attrs map[string]string) string {
	tmpDir := t.TempDir()

	// Create fake vnicc sysfs path: <tmpDir>/devices/qeth/<busID>/vnicc/
	vniccPath := filepath.Join(tmpDir, "devices", "qeth", busID, "vnicc")
	if err := os.MkdirAll(vniccPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Write default attribute files then override with provided attrs
	defaults := map[string]string{
		"flooding":          "0",
		"mcast_flooding":    "0",
		"learning":          "0",
		"learning_timeout":  "600",
		"rx_bcast":          "1",
		"takeover_setvmac":  "0",
		"takeover_learning": "0",
		"bridge_invisible":  "0",
	}
	maps.Copy(defaults, attrs)
	for k, v := range defaults {
		if err := os.WriteFile(filepath.Join(vniccPath, k), []byte(v), filePerm); err != nil {
			t.Fatal(err)
		}
	}

	// Create fake net symlink: <tmpDir>/class/net/<ifaceName>/device -> <tmpDir>/devices/qeth/<busID>
	netPath := filepath.Join(tmpDir, "class", "net", ifaceName)
	if err := os.MkdirAll(netPath, 0755); err != nil {
		t.Fatal(err)
	}
	qethDevicePath := filepath.Join(tmpDir, "devices", "qeth", busID)
	linkPath := filepath.Join(netPath, "device")
	if err := os.Symlink(qethDevicePath, linkPath); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

var _ = Describe("VniccManager", func() {
	var manager *qeth.Manager

	BeforeEach(func() {
		manager = qeth.NewManager(logr.Discard())
	})

	Describe("Apply", func() {
		Context("when applying flooding, mcast_flooding and learning", func() {
			It("should write correct sysfs values in correct order", func() {
				tmpDir := fakeQethEnv(GinkgoT(), "0.0.0220", "enc220", nil)

				trueVal := true
				timeout := 300
				cfg := qeth.VniccConfig{
					Flooding:        &trueVal,
					McastFlooding:   &trueVal,
					Learning:        &trueVal,
					LearningTimeout: &timeout,
				}

				err := manager.ApplyWithBasePath(tmpDir, "enc220", cfg)
				Expect(err).NotTo(HaveOccurred())

				vniccBase := filepath.Join(tmpDir, "devices", "qeth", "0.0.0220", "vnicc")
				Expect(readSysfs(vniccBase, "flooding")).To(Equal("1"))
				Expect(readSysfs(vniccBase, "mcast_flooding")).To(Equal("1"))
				Expect(readSysfs(vniccBase, "learning")).To(Equal("1"))
				Expect(readSysfs(vniccBase, "learning_timeout")).To(Equal("300"))
			})
		})

		Context("when interface is not a qeth device", func() {
			It("should return an error", func() {
				trueVal := true
				cfg := qeth.VniccConfig{Flooding: &trueVal}
				err := manager.ApplyWithBasePath("/nonexistent", "eth0", cfg)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Read", func() {
		It("should read vnicc attributes correctly from sysfs", func() {
			tmpDir := fakeQethEnv(GinkgoT(), "0.0.0420", "enc420", map[string]string{
				"flooding":       "1",
				"mcast_flooding": "1",
				"learning":       "0",
			})

			cfg, err := manager.ReadWithBasePath(tmpDir, "enc420")
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())
			Expect(*cfg.Flooding).To(BeTrue())
			Expect(*cfg.McastFlooding).To(BeTrue())
			Expect(*cfg.Learning).To(BeFalse())
		})

		It("should return an error for non-qeth interfaces", func() {
			tmpDir := GinkgoT().TempDir()
			// Create interface dir without qeth device symlink
			netPath := filepath.Join(tmpDir, "class", "net", "eth0")
			Expect(os.MkdirAll(netPath, 0755)).To(Succeed())

			cfg, err := manager.ReadWithBasePath(tmpDir, "eth0")
			Expect(err).To(HaveOccurred()) // symlink missing → bus ID resolution fails
			Expect(cfg).To(BeNil())
		})
	})

	Describe("IsQethInterface", func() {
		It("should return true for qeth interfaces", func() {
			tmpDir := fakeQethEnv(GinkgoT(), "0.0.0220", "enc220", nil)
			Expect(manager.IsQethInterfaceWithBasePath(tmpDir, "enc220")).To(BeTrue())
		})

		It("should return false for non-qeth interfaces", func() {
			Expect(manager.IsQethInterfaceWithBasePath("/nonexistent", "eth0")).To(BeFalse())
		})
	})
})

func readSysfs(base, attr string) string {
	data, err := os.ReadFile(filepath.Join(base, attr))
	if err != nil {
		return ""
	}
	return string(data)
}
