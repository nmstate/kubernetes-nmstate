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

package enactmentstatus

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Error messages formatting", func() {
	var (
		debugInvalidInput      = "      2021-04-15 08:17:35,057 root         DEBUG    Async action: Create checkpoint started\n      2021-04-15 08:17:35,060 root         DEBUG    Checkpoint None created for all devices\n"
		debugValidInput        = "A message containing debug that should not be removed.\n"
		tracebackInvalidInput  = "      Traceback (most recent call last):\n        File \"/usr/bin/nmstatectl\", line 11, in <module>\n          load_entry_point('nmstate==0.3.6', 'console_scripts', 'nmstatectl')()\n      File \"/usr/lib/python3.6/site-packages/nmstatectl/nmstatectl.py\", line 69, in main\n     f\"Interface {iface.name} has unknown slave: \"\n      libnmstate.error.NmstateValueError: Interface bond1 has unknown slave: eth10\n      "
		tracebackInvalidOutput = "      libnmstate.error.NmstateValueError\n  Interface bond1 has unknown slave\n    eth10\n"
		tracebackValidInput    = "A message containing File \"/usr/bin/nmstatectl\" that should not be removed.\n"
		failedToExecuteInput   = " failed to execute nmstatectl set --no-commit --timeout 480: 'exit status 1' '' '2021-02-22 11:10:08,962 root         WARNING  libnm version 1.26.7 mismatches NetworkManager version 1.29.9\n"
		failedToExecuteOutput  = "failed to execute nmstatectl set --no-commit --timeout 480: 'exit status 1'\n"
		pingInvalidInput       = "rolling back desired state configuration: failed runnig probes after network changes: failed runnig probe 'ping' with after network reconfiguration -> currentState: ---\n      dns-resolver:\n      config:\n          search: []\n          server: []\n        running: {}\n      route-rules:\n        config: []\n      : failed to retrieve default gw at runProbes: timed out waiting for the condition"
		pingInvalidOutput      = "      \n  failed to retrieve default gw at runProbes\n    timed out waiting for the condition\n"
		pingValidInput         = "rolling back desired state configuration: failed runnig probes after network changes: failed runnig probe 'ping' with after network reconfiguration.\nThe rest of the message should be kept.\n"
		pingValidOutput        = "rolling back desired state configuration\n  failed runnig probes after network changes\n    failed runnig probe 'ping' with after network reconfiguration.\nThe rest of the message should be kept.\n"
		desiredStateYaml       = "libnmstate.error.NmstateVerificationError:\n      desired\n      =======\n---\n      name: eth1\n      type: ethernet\n      state: up\n"
	)

	Context("With DEBUG text", func() {
		It("Should remove DEBUG message", func() {
			Expect(FormatErrorString(debugInvalidInput)).To(Equal(""))
		})
		It("Should keep message with debug keyword", func() {
			Expect(FormatErrorString(debugValidInput)).To(Equal(debugValidInput))
		})
	})

	Context("With Traceback text", func() {
		It("Should remove python traceback", func() {
			Expect(FormatErrorString(tracebackInvalidInput)).To(Equal(tracebackInvalidOutput))
		})
		It("Should keep message with File keyword", func() {
			Expect(FormatErrorString(tracebackValidInput)).To(Equal(tracebackValidInput))
		})
	})

	Context("With failed to execute text", func() {
		It("Should remove warning form the line", func() {
			Expect(FormatErrorString(failedToExecuteInput)).To(Equal(failedToExecuteOutput))
		})
	})

	Context("With network reconfiguration text", func() {
		It("Should remove yaml", func() {
			Expect(FormatErrorString(pingInvalidInput)).To(Equal(pingInvalidOutput))
		})
		It("Should keep message", func() {
			Expect(FormatErrorString(pingValidInput)).To(Equal(pingValidOutput))
		})
	})
	Context("With yaml states", func() {
		It("Should keep the original message", func() {
			Expect(FormatErrorString(desiredStateYaml)).To(Equal(desiredStateYaml))
		})
	})

	Describe("Error messages compression and decompression", func() {
		var (
			messages = []string{
				"A message containing debug that should not be removed.\n",
				"      libnmstate.error.NmstateValueError\n  Interface bond1 has unknown slave\n    eth10\n",
				"A message containing File \"/usr/bin/nmstatectl\" that should not be removed.\n",
				"failed to execute nmstatectl set --no-commit --timeout 480: 'exit status 1'\n",
				"      \n  failed to retrieve default gw at runProbes\n    timed out waiting for the condition\n",
			}
		)
		Context("With a sample message", func() {
			It("Should decompress the message correctly", func() {
				for _, message := range messages {
					encodedMessage := CompressAndEncodeMessage(message)
					Expect(message).To(Equal(DecodeAndDecompressMessage(encodedMessage)))
				}
			})
		})
	})
})
