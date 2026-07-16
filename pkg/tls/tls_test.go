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
	cryptotls "crypto/tls"

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

var _ = Describe("GetTLSProfileSpec strictness", func() {
	It("errors on unknown profile type instead of falling back", func() {
		_, err := GetTLSProfileSpec(&tlsSecurityProfile{profileType: TLSProfileType("Bogus")})
		Expect(err).To(MatchError(ErrUnknownProfileType))
	})

	It("errors on Custom profile with nil spec", func() {
		_, err := GetTLSProfileSpec(&tlsSecurityProfile{profileType: TLSProfileCustomType})
		Expect(err).To(MatchError(ErrCustomProfileNil))
	})
})

var _ = Describe("NewTLSConfigFromProfile", func() {
	It("applies min version, ciphers and curves", func() {
		opts, unsupported, err := NewTLSConfigFromProfile(TLSProfileSpec{
			MinTLSVersion: VersionTLS12,
			Ciphers: []string{
				"TLS_AES_128_GCM_SHA256",
				"TLS_AES_256_GCM_SHA384",
				"TLS_CHACHA20_POLY1305_SHA256",
				"ECDHE-RSA-AES128-GCM-SHA256",
			},
			Curves: []string{"X25519MLKEM768", "X25519"},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(unsupported.IsEmpty()).To(BeTrue())
		conf := &cryptotls.Config{}
		opts(conf)
		Expect(conf.MinVersion).To(Equal(uint16(cryptotls.VersionTLS12)))
		Expect(conf.CipherSuites).To(Equal([]uint16{
			cryptotls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		}), "CipherSuites must only carry TLS 1.2 and older suite IDs")
		Expect(conf.CurvePreferences).To(Equal([]cryptotls.CurveID{cryptotls.X25519MLKEM768, cryptotls.X25519}))
	})

	It("serves TLS 1.3 only when the profile permits no TLS 1.2 ciphers", func() {
		opts, unsupported, err := NewTLSConfigFromProfile(TLSProfileSpec{
			MinTLSVersion: VersionTLS12,
			Ciphers: []string{
				"TLS_AES_128_GCM_SHA256",
				"TLS_AES_256_GCM_SHA384",
				"TLS_CHACHA20_POLY1305_SHA256",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(unsupported.IsEmpty()).To(BeTrue())
		conf := &cryptotls.Config{}
		opts(conf)
		Expect(conf.MinVersion).To(Equal(uint16(cryptotls.VersionTLS13)))
		Expect(conf.CipherSuites).To(BeNil())
	})

	It("reports TLS 1.3 cipher restrictions Go cannot enforce", func() {
		_, unsupported, err := NewTLSConfigFromProfile(TLSProfileSpec{
			MinTLSVersion: VersionTLS12,
			Ciphers:       []string{"ECDHE-RSA-AES128-GCM-SHA256", "TLS_AES_128_GCM_SHA256"},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(unsupported.IsEmpty()).To(BeFalse())
		Expect(unsupported.UnenforceableCiphers).To(Equal([]string{
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
		}))
		Expect(unsupported.Message()).To(ContainSubstring("TLS_AES_256_GCM_SHA384"))
		Expect(unsupported.Message()).To(ContainSubstring("cannot restrict TLS 1.3"))
	})

	It("does not flag unenforceable ciphers for the predefined profiles", func() {
		for _, profileType := range []TLSProfileType{TLSProfileOldType, TLSProfileIntermediateType, TLSProfileModernType} {
			_, unsupported, err := NewTLSConfigFromProfile(*TLSProfiles[profileType])
			Expect(err).NotTo(HaveOccurred())
			Expect(unsupported.IsEmpty()).To(BeTrue(), "profile %s should be fully enforceable", profileType)
		}
	})

	It("leaves CurvePreferences nil when profile has no curves", func() {
		opts, _, err := NewTLSConfigFromProfile(*TLSProfiles[TLSProfileIntermediateType])
		Expect(err).NotTo(HaveOccurred())
		conf := &cryptotls.Config{}
		opts(conf)
		Expect(conf.CurvePreferences).To(BeNil())
	})

	It("reports partially unsupported entries but still succeeds", func() {
		_, unsupported, err := NewTLSConfigFromProfile(TLSProfileSpec{
			MinTLSVersion: VersionTLS12,
			Ciphers:       []string{"ECDHE-RSA-AES128-GCM-SHA256", "NOT-A-CIPHER"},
			Curves:        []string{"X25519", "brainpoolP256r1"},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(unsupported.Ciphers).To(Equal([]string{"NOT-A-CIPHER"}))
		Expect(unsupported.Curves).To(Equal([]string{"brainpoolP256r1"}))
		Expect(unsupported.Message()).To(ContainSubstring("NOT-A-CIPHER"))
		Expect(unsupported.Message()).To(ContainSubstring("brainpoolP256r1"))
	})

	It("fails closed when no cipher is supported below TLS 1.3", func() {
		_, _, err := NewTLSConfigFromProfile(TLSProfileSpec{
			MinTLSVersion: VersionTLS12,
			Ciphers:       []string{"NOT-A-CIPHER"},
		})
		Expect(err).To(MatchError(ErrNoSupportedCiphers))
	})

	It("does not fail on unsupported ciphers with TLS 1.3 minimum", func() {
		_, _, err := NewTLSConfigFromProfile(TLSProfileSpec{
			MinTLSVersion: VersionTLS13,
			Ciphers:       []string{"NOT-A-CIPHER"},
		})
		Expect(err).NotTo(HaveOccurred())
	})

	It("fails closed when no curve is supported", func() {
		_, _, err := NewTLSConfigFromProfile(TLSProfileSpec{
			MinTLSVersion: VersionTLS13,
			Curves:        []string{"brainpoolP256r1"},
		})
		Expect(err).To(MatchError(ErrNoSupportedCurves))
	})

	It("fails closed on unknown minimum TLS version", func() {
		_, _, err := NewTLSConfigFromProfile(TLSProfileSpec{
			MinTLSVersion: TLSProtocolVersion("VersionTLS99"),
		})
		Expect(err).To(HaveOccurred())
	})
})
