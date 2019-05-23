package nmstate_tests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Reporting State", func() {
	It("should report correct node name", func() {
		var obtainedState *nmstatev1.NodeNetworkState
		Eventually(func() (*nmstatev1.NodeNetworkState, error) {
			var err error
			obtainedState, err = nmstateNNSs.Get(firstNodeName, metav1.GetOptions{})
			return obtainedState, err
		}).ShouldNot(BeNil())
		//FIXME: We have to replace this with gomega/gstruct directly at Eventually
		//       matcher, haven't being able to make it work
		Expect(obtainedState.Spec.NodeName).To(Equal(firstNodeName))
	})
})
