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

func apiServerObj(tlsSecurityProfile map[string]any) map[string]any {
	spec := map[string]any{}
	if tlsSecurityProfile != nil {
		spec["tlsSecurityProfile"] = tlsSecurityProfile
	}
	return map[string]any{
		"apiVersion": "config.openshift.io/v1",
		"kind":       "APIServer",
		"metadata":   map[string]any{"name": "cluster"},
		"spec":       spec,
	}
}

var _ = Describe("parseTLSSecurityProfile", func() {
	It("parses custom profile curves", func() {
		profile, err := parseTLSSecurityProfile(apiServerObj(map[string]any{
			"type": "Custom",
			"custom": map[string]any{
				"minTLSVersion": "VersionTLS12",
				"ciphers":       []any{"TLS_AES_128_GCM_SHA256"},
				"curves":        []any{"X25519MLKEM768", "X25519"},
			},
		}))
		Expect(err).NotTo(HaveOccurred())
		Expect(profile.customSpec.Curves).To(Equal([]string{"X25519MLKEM768", "X25519"}))
	})

	It("leaves curves nil when absent", func() {
		profile, err := parseTLSSecurityProfile(apiServerObj(map[string]any{
			"type": "Custom",
			"custom": map[string]any{
				"minTLSVersion": "VersionTLS12",
				"ciphers":       []any{"TLS_AES_128_GCM_SHA256"},
			},
		}))
		Expect(err).NotTo(HaveOccurred())
		Expect(profile.customSpec.Curves).To(BeNil())
	})
})

var _ = Describe("GetTLSProfileSpec", func() {
	It("defaults to Intermediate when no profile is set", func() {
		spec, err := GetTLSProfileSpec(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(spec).To(Equal(*TLSProfiles[TLSProfileIntermediateType]))
	})

	It("returns named profiles without curves", func() {
		spec, err := GetTLSProfileSpec(&tlsSecurityProfile{profileType: TLSProfileModernType})
		Expect(err).NotTo(HaveOccurred())
		Expect(spec.MinTLSVersion).To(Equal(VersionTLS13))
		Expect(spec.Curves).To(BeEmpty())
	})
})
