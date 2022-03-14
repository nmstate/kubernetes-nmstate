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

package nodenetworkconfigurationpolicy

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	jsonpatch "github.com/evanphx/json-patch"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NNCP Webhook Test Suite")
}

func requestForPolicy(policy nmstatev1.NodeNetworkConfigurationPolicy) webhook.AdmissionRequest {
	data, err := json.Marshal(policy)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	request := webhook.AdmissionRequest{}
	request.Object = runtime.RawExtension{
		Raw: data,
	}
	return request
}

func patchPolicy(
	policy nmstatev1.NodeNetworkConfigurationPolicy,
	response webhook.AdmissionResponse,
) nmstatev1.NodeNetworkConfigurationPolicy {
	patch, err := jsonpatch.DecodePatch(response.Patch)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	old, err := json.Marshal(policy)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	modified, err := patch.Apply(old)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	policy = nmstatev1.NodeNetworkConfigurationPolicy{}
	err = json.Unmarshal(modified, &policy)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return policy
}
