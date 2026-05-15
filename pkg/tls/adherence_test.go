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

package tls

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ShouldHonorClusterTLSProfile", func() {
	DescribeTable("returns the correct decision for each adherence value",
		func(adherence TLSAdherencePolicy, expected bool) {
			Expect(ShouldHonorClusterTLSProfile(adherence)).To(Equal(expected))
		},
		Entry("empty (no opinion) returns false", TLSAdherencePolicy(""), false),
		Entry("LegacyAdheringComponentsOnly returns false",
			TLSAdherenceLegacyAdheringComponentsOnly, false),
		Entry("StrictAllComponents returns true",
			TLSAdherenceStrictAllComponents, true),
		Entry("unknown future value returns true (forward compatibility)",
			TLSAdherencePolicy("SomeFutureValue"), true),
		Entry("arbitrary mixed-case unknown value returns true",
			TLSAdherencePolicy("strictallcomponents"), true),
	)
})

var _ = Describe("parseTLSAdherence", func() {
	It("returns empty when spec.tlsAdherence is missing", func() {
		obj := map[string]interface{}{
			"spec": map[string]interface{}{},
		}
		Expect(parseTLSAdherence(obj)).To(Equal(TLSAdherencePolicy("")))
	})

	It("returns empty when spec is entirely missing", func() {
		obj := map[string]interface{}{}
		Expect(parseTLSAdherence(obj)).To(Equal(TLSAdherencePolicy("")))
	})

	It("returns the value when spec.tlsAdherence is set", func() {
		obj := map[string]interface{}{
			"spec": map[string]interface{}{
				"tlsAdherence": "StrictAllComponents",
			},
		}
		Expect(parseTLSAdherence(obj)).To(Equal(TLSAdherenceStrictAllComponents))
	})

	It("returns empty when spec.tlsAdherence is not a string", func() {
		obj := map[string]interface{}{
			"spec": map[string]interface{}{
				"tlsAdherence": 42,
			},
		}
		// unstructured.NestedString reports an error in this case; we treat
		// it as the zero value to keep the operator resilient against
		// unexpected cluster state.
		Expect(parseTLSAdherence(obj)).To(Equal(TLSAdherencePolicy("")))
	})

	It("preserves an unknown adherence value verbatim", func() {
		obj := map[string]interface{}{
			"spec": map[string]interface{}{
				"tlsAdherence": "SomeFutureValue",
			},
		}
		Expect(parseTLSAdherence(obj)).To(Equal(TLSAdherencePolicy("SomeFutureValue")))
	})
})
