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
	"encoding/json"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	shared "github.com/nmstate/kubernetes-nmstate/api/shared"
)

type fakeVniccManager struct {
	applyCalled bool
	lastIface   string
	lastCfg     VniccConfig
}

func (f *fakeVniccManager) Apply(
	ifaceName string,
	cfg VniccConfig,
) error {
	f.applyCalled = true
	f.lastIface = ifaceName
	f.lastCfg = cfg
	return nil
}

var _ = Describe("VniccHook", func() {
	var (
		manager *fakeVniccManager
		hook    *VniccHook
	)

	BeforeEach(func() {
		manager = &fakeVniccManager{}

		hook = NewVniccHook(logr.Discard())

		// override real sysfs applier with fake
		hook.applier = manager
	})

	It("should apply and strip qeth.vnicc config", func() {
		rawJSON := []byte(`{
			"interfaces": [
				{
					"name": "enc220",
					"type": "ethernet",
					"qeth": {
						"vnicc": {
							"flooding": true
						}
					}
				}
			]
		}`)

		cleanedJSON, err := hook.processJSON(rawJSON)

		Expect(err).NotTo(HaveOccurred())

		Expect(manager.applyCalled).To(BeTrue())
		Expect(manager.lastIface).To(Equal("enc220"))

		var cleaned map[string]any

		Expect(json.Unmarshal(cleanedJSON, &cleaned)).To(Succeed())

		ifaces := cleaned["interfaces"].([]any)
		iface := ifaces[0].(map[string]any)

		qethMap, ok := iface["qeth"].(map[string]any)

		if ok {
			_, exists := qethMap["vnicc"]
			Expect(exists).To(BeFalse())
		}
	})

	It("should leave interfaces without qeth unchanged", func() {
		rawJSON := []byte(`{
			"interfaces": [
				{
					"name": "eth0",
					"type": "ethernet"
				}
			]
		}`)

		cleanedJSON, err := hook.processJSON(rawJSON)

		Expect(err).NotTo(HaveOccurred())
		Expect(cleanedJSON).To(MatchJSON(rawJSON))
		Expect(manager.applyCalled).To(BeFalse())
	})

	It("should fail when interface name is missing", func() {
		rawJSON := []byte(`{
			"interfaces": [
				{
					"type": "ethernet",
					"qeth": {
						"vnicc": {
							"flooding": true
						}
					}
				}
			]
		}`)

		_, err := hook.processJSON(rawJSON)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(
			ContainSubstring("missing 'name' field"),
		)
	})

	It("should process shared.State safely", func() {
		stateJSON := `{
			"interfaces": [
				{
					"name": "eth0",
					"type": "ethernet"
				}
			]
		}`

		state := shared.State{
			Raw: []byte(stateJSON),
		}

		_, err := hook.ProcessAndStrip(state)

		// On non-s390x this is no-op
		// On s390x this still succeeds
		Expect(err).NotTo(HaveOccurred())
	})
})
