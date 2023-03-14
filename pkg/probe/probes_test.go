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

package probe

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("defaultGw", func() {
	var (
		currentState      string
		expectedDefaultGw string
		expectedError     error
	)

	Context("when there is a single default route", func() {
		BeforeEach(func() {
			currentState = `
routes:
  config: []
  running:
  - destination: 9.0.0.0/24
    metric: 150
    next-hop-address: 192.168.123.1
    next-hop-interface: eth1
    table-id: 254
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`
			expectedDefaultGw = "192.168.66.2"
			expectedError = nil
		})
		It("should return the default gateway", func() {
			nmstatectlShow = func() (string, error) {
				return currentState, nil
			}
			returnedDefaultGw, err := defaultGw()
			if expectedError != nil {
				Expect(err).To(Equal(expectedError))
				return
			}
			Expect(err).To(BeNil())
			Expect(returnedDefaultGw).To(Equal(expectedDefaultGw))
		})
	})

	Context("when there are multiple default routes in the main table", func() {
		BeforeEach(func() {
			currentState = `
routes:
  config: []
  running:
  - destination: 9.0.0.0/24
    metric: 150
    next-hop-address: 192.168.123.1
    next-hop-interface: eth1
    table-id: 254
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.77.3
    next-hop-interface: eth1
    table-id: 254
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`
			expectedDefaultGw = "192.168.77.3"
			expectedError = nil
		})
		It("should return the first default gateway it finds", func() {
			nmstatectlShow = func() (string, error) {
				return currentState, nil
			}
			returnedDefaultGw, err := defaultGw()
			if expectedError != nil {
				Expect(err).To(Equal(expectedError))
				return
			}
			Expect(err).To(BeNil())
			Expect(returnedDefaultGw).To(Equal(expectedDefaultGw))
		})
	})

	Context("when there are default routes in other tables than the default table", func() {
		BeforeEach(func() {
			currentState = `
routes:
  config: []
  running:
  - destination: 9.0.0.0/24
    metric: 150
    next-hop-address: 192.168.123.1
    next-hop-interface: eth1
    table-id: 254
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.77.3
    next-hop-interface: eth1
    table-id: 123
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`
			expectedDefaultGw = "192.168.66.2"
			expectedError = nil
		})
		It("should return the default gateway from the main table", func() {
			nmstatectlShow = func() (string, error) {
				return currentState, nil
			}
			returnedDefaultGw, err := defaultGw()
			if expectedError != nil {
				Expect(err).To(Equal(expectedError))
				return
			}
			Expect(err).To(BeNil())
			Expect(returnedDefaultGw).To(Equal(expectedDefaultGw))
		})
	})

	Context("when the table-id of the default route is unset", func() {
		BeforeEach(func() {
			currentState = `
routes:
  config: []
  running:
  - destination: 9.0.0.0/24
    metric: 150
    next-hop-address: 192.168.123.1
    next-hop-interface: eth1
    table-id: 254
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.77.3
    next-hop-interface: eth1
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`
			expectedDefaultGw = "192.168.77.3"
			expectedError = nil
		})
		It("should return the default gateway", func() {
			nmstatectlShow = func() (string, error) {
				return currentState, nil
			}
			returnedDefaultGw, err := defaultGw()
			if expectedError != nil {
				Expect(err).To(Equal(expectedError))
				return
			}
			Expect(err).To(BeNil())
			Expect(returnedDefaultGw).To(Equal(expectedDefaultGw))
		})
	})

	Context("when the table-id of the default route is 0", func() {
		BeforeEach(func() {
			currentState = `
routes:
  config: []
  running:
  - destination: 9.0.0.0/24
    metric: 150
    next-hop-address: 192.168.123.1
    next-hop-interface: eth1
    table-id: 254
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.77.3
    next-hop-interface: eth1
    table-id: 0
  - destination: 0.0.0.0/0
    metric: 102
    next-hop-address: 192.168.66.2
    next-hop-interface: eth1
    table-id: 254
`
			expectedDefaultGw = "192.168.77.3"
			expectedError = nil
		})
		It("should return the default gateway", func() {
			nmstatectlShow = func() (string, error) {
				return currentState, nil
			}
			returnedDefaultGw, err := defaultGw()
			if expectedError != nil {
				Expect(err).To(Equal(expectedError))
				return
			}
			Expect(err).To(BeNil())
			Expect(returnedDefaultGw).To(Equal(expectedDefaultGw))
		})
	})
})
