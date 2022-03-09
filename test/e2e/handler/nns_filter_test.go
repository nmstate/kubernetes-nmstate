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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nmstate/kubernetes-nmstate/pkg/state"
	"github.com/nmstate/kubernetes-nmstate/test/cmd"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("[nns] NNS Interface filter", func() {
	BeforeEach(func() {
		// Make sure NNSes are present
		for _, node := range nodes {
			key := types.NamespacedName{Name: node}
			_ = nodeNetworkState(key)
		}
	})
	It("should not log errors related to NNS interface filtering", func() {
		combinedHandlerLogs, err := cmd.Kubectl("logs", "-lname=nmstate-handler", "-n", "nmstate")
		Expect(err).ToNot(HaveOccurred())
		Expect(combinedHandlerLogs).ToNot(ContainSubstring(state.InterfaceFilter))
	})
})
