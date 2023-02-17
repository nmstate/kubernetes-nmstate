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

package handler

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nmstate/kubernetes-nmstate/test/e2e/policy"
	"github.com/nmstate/kubernetes-nmstate/test/runner"
)

var _ = Describe("checkpoin", func() {
	Context("when is not committed from previous operation", func() {
		BeforeEach(func() {
			stateAsJSON, err := linuxBrUpNoPorts(bridge1).MarshalJSON()
			Expect(err).ToNot(HaveOccurred())
			runner.RunAtHandlerPods("bash", "-c", fmt.Sprintf("echo '%s' | nmstatectl apply --no-commit --timeout 60", string(stateAsJSON)))
		})
		Context("and new nncp is configured", func() {
			BeforeEach(func() {
				updateDesiredState(linuxBrUpNoPorts(bridge1))
			})
			AfterEach(func() {
				updateDesiredStateAndWait(linuxBrAbsent(bridge1))
			})
			It("should remove pending checkpoint and continue", func() {
				policy.WaitForAvailableTestPolicy()
			})
		})
	})
})
