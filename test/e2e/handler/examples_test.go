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

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	example "github.com/nmstate/kubernetes-nmstate/test/doc"
)

// This suite checks that all examples in our docs can be successfully applied.
// It only checks the top level API, hence it does not verify that the
// configuration is indeed applied on nodes. That should be tested by dedicated
// test suites for each feature.
var _ = Describe("[user-guide] Examples", func() {

	beforeTestIfaceExample := func(fileName string) {
		kubectlAndCheck("apply", "-f", fmt.Sprintf("docs/examples/%s", fileName))
	}

	testIfaceExample := func(policyName string) {
		kubectlAndCheck("wait", "nncp", policyName, "--for", "condition=Available", "--timeout", "2m")
	}

	afterIfaceExample := func(policyName string, ifaceNames []string, cleanupState *nmstate.State) {
		deletePolicy(policyName)

		if len(ifaceNames) > 0 {
			for _, ifaceName := range ifaceNames {
				updateDesiredStateAndWait(interfaceAbsent(ifaceName))
			}
		}

		if cleanupState != nil {
			updateDesiredStateAndWait(*cleanupState)
		}

		resetDesiredStateForNodes()
	}

	for _, e := range example.ExampleSpecs() {
		example := e
		Context(e.Name, func() {
			BeforeEach(func() {
				beforeTestIfaceExample(example.FileName)
			})

			AfterEach(func() {
				afterIfaceExample(example.PolicyName, example.IfaceNames, example.CleanupState)
			})

			It("should succeed applying the policy", func() {
				testIfaceExample(example.PolicyName)
			})
		})
	}
})
