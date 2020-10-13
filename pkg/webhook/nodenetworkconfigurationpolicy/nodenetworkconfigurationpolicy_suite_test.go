package nodenetworkconfigurationpolicy

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	jsonpatch "github.com/evanphx/json-patch"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

func TestUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "NNCP Webhook Test Suite")
}

func requestForPolicy(policy nmstatev1beta1.NodeNetworkConfigurationPolicy) webhook.AdmissionRequest {
	data, err := json.Marshal(policy)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	request := webhook.AdmissionRequest{}
	request.Object = runtime.RawExtension{
		Raw: data,
	}
	return request
}
func patchPolicy(policy nmstatev1beta1.NodeNetworkConfigurationPolicy, response webhook.AdmissionResponse) nmstatev1beta1.NodeNetworkConfigurationPolicy {

	patch, err := jsonpatch.DecodePatch(response.Patch)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	old, err := json.Marshal(policy)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	modified, err := patch.Apply(old)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	policy = nmstatev1beta1.NodeNetworkConfigurationPolicy{}
	err = json.Unmarshal(modified, &policy)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return policy
}
