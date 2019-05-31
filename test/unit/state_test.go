package unit

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"gopkg.in/yaml.v2"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"
)

var _ = Describe("State", func() {
	var (
		obtained nmstatev1.NodeNetworkState
		oneIf    = `apiVersion: nmstate.io/v1
kind: NodeNetworkState
metadata:
  name: node01
spec:
  managed: true
  nodeName: node01
  desiredState:
    interfaces:
      - name: eth1
        type: ethernet
        state: up
`
	)
	BeforeEach(func() {
		err := yaml.Unmarshal([]byte(oneIf), &obtained)
		Expect(err).ToNot(HaveOccurred())

	})
	Context("with one interface", func() {
		It("should not be nil", func() {
			fmt.Printf("%+v\n", obtained)
			Expect(obtained.Spec.DesiredState).ToNot(BeNil())
		})
	})
})
