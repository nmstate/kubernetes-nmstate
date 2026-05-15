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
	"context"
	"crypto/tls"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// newAPIServerUnstructured returns an *unstructured.Unstructured representing
// a config.openshift.io/v1 APIServer object named "cluster". The caller can
// further mutate obj.Object["spec"] before seeding into the fake client.
func newAPIServerUnstructured() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(apiServerGVK)
	obj.SetName(apiServerName)
	obj.Object["spec"] = map[string]interface{}{}
	return obj
}

// setTLSSecurityProfile installs spec.tlsSecurityProfile on obj.
func setTLSSecurityProfile(obj *unstructured.Unstructured, profile map[string]interface{}) {
	spec, _ := obj.Object["spec"].(map[string]interface{})
	if spec == nil {
		spec = map[string]interface{}{}
		obj.Object["spec"] = spec
	}
	spec["tlsSecurityProfile"] = profile
}

// setTLSAdherence installs spec.tlsAdherence on obj.
func setTLSAdherence(obj *unstructured.Unstructured, adherence string) {
	spec, _ := obj.Object["spec"].(map[string]interface{})
	if spec == nil {
		spec = map[string]interface{}{}
		obj.Object["spec"] = spec
	}
	spec["tlsAdherence"] = adherence
}

// newFakeClient builds a controller-runtime fake client seeded with the
// provided unstructured objects.
func newFakeClient(objs ...client.Object) client.Client {
	builder := fake.NewClientBuilder()
	if len(objs) > 0 {
		builder = builder.WithObjects(objs...)
	}
	return builder.Build()
}

var _ = Describe("parseTLSSecurityProfile", func() {
	It("returns (nil, nil) when spec.tlsSecurityProfile is absent", func() {
		obj := map[string]interface{}{
			"spec": map[string]interface{}{},
		}
		profile, err := parseTLSSecurityProfile(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(profile).To(BeNil())
	})

	It("returns (nil, nil) when spec is missing entirely", func() {
		profile, err := parseTLSSecurityProfile(map[string]interface{}{})
		Expect(err).NotTo(HaveOccurred())
		Expect(profile).To(BeNil())
	})

	It("returns a profile with empty profileType when type is missing", func() {
		obj := map[string]interface{}{
			"spec": map[string]interface{}{
				"tlsSecurityProfile": map[string]interface{}{},
			},
		}
		profile, err := parseTLSSecurityProfile(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(profile).NotTo(BeNil())
		Expect(profile.profileType).To(Equal(TLSProfileType("")))
		Expect(profile.customSpec).To(BeNil())
	})

	DescribeTable("returns the right non-custom profile type",
		func(profileType TLSProfileType) {
			obj := map[string]interface{}{
				"spec": map[string]interface{}{
					"tlsSecurityProfile": map[string]interface{}{
						"type": string(profileType),
					},
				},
			}
			profile, err := parseTLSSecurityProfile(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(profile).NotTo(BeNil())
			Expect(profile.profileType).To(Equal(profileType))
			Expect(profile.customSpec).To(BeNil())
		},
		Entry("Old", TLSProfileOldType),
		Entry("Intermediate", TLSProfileIntermediateType),
		Entry("Modern", TLSProfileModernType),
	)

	It("returns a populated customSpec for Custom type with custom block", func() {
		obj := map[string]interface{}{
			"spec": map[string]interface{}{
				"tlsSecurityProfile": map[string]interface{}{
					"type": "Custom",
					"custom": map[string]interface{}{
						"minTLSVersion": "VersionTLS12",
						"ciphers":       []interface{}{"ECDHE-RSA-AES128-GCM-SHA256", "ECDHE-RSA-AES256-GCM-SHA384"},
					},
				},
			},
		}
		profile, err := parseTLSSecurityProfile(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(profile).NotTo(BeNil())
		Expect(profile.profileType).To(Equal(TLSProfileCustomType))
		Expect(profile.customSpec).NotTo(BeNil())
		Expect(profile.customSpec.MinTLSVersion).To(Equal(VersionTLS12))
		Expect(profile.customSpec.Ciphers).To(Equal([]string{"ECDHE-RSA-AES128-GCM-SHA256", "ECDHE-RSA-AES256-GCM-SHA384"}))
	})

	It("returns Custom type with nil customSpec when custom block is absent", func() {
		obj := map[string]interface{}{
			"spec": map[string]interface{}{
				"tlsSecurityProfile": map[string]interface{}{
					"type": "Custom",
				},
			},
		}
		profile, err := parseTLSSecurityProfile(obj)
		Expect(err).NotTo(HaveOccurred())
		Expect(profile).NotTo(BeNil())
		Expect(profile.profileType).To(Equal(TLSProfileCustomType))
		Expect(profile.customSpec).To(BeNil())
	})

	It("returns an error when spec.tlsSecurityProfile is the wrong type", func() {
		obj := map[string]interface{}{
			"spec": map[string]interface{}{
				"tlsSecurityProfile": "not-a-map",
			},
		}
		_, err := parseTLSSecurityProfile(obj)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("GetTLSProfileSpec", func() {
	It("returns the default Intermediate profile when profile is nil", func() {
		spec, err := GetTLSProfileSpec(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(spec).To(Equal(*TLSProfiles[TLSProfileIntermediateType]))
	})

	It("returns the default Intermediate profile when profileType is empty", func() {
		spec, err := GetTLSProfileSpec(&tlsSecurityProfile{})
		Expect(err).NotTo(HaveOccurred())
		Expect(spec).To(Equal(*TLSProfiles[TLSProfileIntermediateType]))
	})

	DescribeTable("returns the canned profile for a known non-custom type",
		func(profileType TLSProfileType) {
			spec, err := GetTLSProfileSpec(&tlsSecurityProfile{profileType: profileType})
			Expect(err).NotTo(HaveOccurred())
			Expect(spec).To(Equal(*TLSProfiles[profileType]))
		},
		Entry("Old", TLSProfileOldType),
		Entry("Intermediate", TLSProfileIntermediateType),
		Entry("Modern", TLSProfileModernType),
	)

	It("falls back to Intermediate when the non-custom type is unknown", func() {
		spec, err := GetTLSProfileSpec(&tlsSecurityProfile{profileType: "ImaginaryProfile"})
		Expect(err).NotTo(HaveOccurred())
		Expect(spec).To(Equal(*TLSProfiles[TLSProfileIntermediateType]))
	})

	It("returns the customSpec verbatim for Custom type", func() {
		custom := TLSProfileSpec{
			Ciphers:       []string{"ECDHE-RSA-AES128-GCM-SHA256"},
			MinTLSVersion: VersionTLS13,
		}
		spec, err := GetTLSProfileSpec(&tlsSecurityProfile{
			profileType: TLSProfileCustomType,
			customSpec:  &custom,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(spec).To(Equal(custom))
	})

	It("returns ErrCustomProfileNil when Custom type lacks customSpec", func() {
		_, err := GetTLSProfileSpec(&tlsSecurityProfile{profileType: TLSProfileCustomType})
		Expect(errors.Is(err, ErrCustomProfileNil)).To(BeTrue())
	})
})

var _ = Describe("FetchAPIServerTLSConfig", func() {
	ctx := context.Background()

	It("returns a wrapped error when the APIServer is not found", func() {
		c := newFakeClient()
		_, err := FetchAPIServerTLSConfig(ctx, c)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to get APIServer"))
		Expect(err.Error()).To(ContainSubstring(apiServerName))
	})

	It("returns the default Intermediate profile and empty adherence when spec is empty", func() {
		obj := newAPIServerUnstructured()
		cfg, err := FetchAPIServerTLSConfig(ctx, newFakeClient(obj))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Profile).To(Equal(*TLSProfiles[TLSProfileIntermediateType]))
		Expect(cfg.Adherence).To(Equal(TLSAdherencePolicy("")))
	})

	It("returns the configured non-custom profile and adherence", func() {
		obj := newAPIServerUnstructured()
		setTLSSecurityProfile(obj, map[string]interface{}{"type": "Intermediate"})
		setTLSAdherence(obj, "StrictAllComponents")
		cfg, err := FetchAPIServerTLSConfig(ctx, newFakeClient(obj))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Profile).To(Equal(*TLSProfiles[TLSProfileIntermediateType]))
		Expect(cfg.Adherence).To(Equal(TLSAdherenceStrictAllComponents))
	})

	It("returns the custom profile spec verbatim", func() {
		obj := newAPIServerUnstructured()
		setTLSSecurityProfile(obj, map[string]interface{}{
			"type": "Custom",
			"custom": map[string]interface{}{
				"minTLSVersion": "VersionTLS13",
				"ciphers":       []interface{}{"TLS_AES_128_GCM_SHA256"},
			},
		})
		cfg, err := FetchAPIServerTLSConfig(ctx, newFakeClient(obj))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Profile.MinTLSVersion).To(Equal(VersionTLS13))
		Expect(cfg.Profile.Ciphers).To(Equal([]string{"TLS_AES_128_GCM_SHA256"}))
	})

	It("returns ErrCustomProfileNil when Custom type lacks the custom block", func() {
		obj := newAPIServerUnstructured()
		setTLSSecurityProfile(obj, map[string]interface{}{"type": "Custom"})
		_, err := FetchAPIServerTLSConfig(ctx, newFakeClient(obj))
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, ErrCustomProfileNil)).To(BeTrue())
	})
})

var _ = Describe("NewTLSConfigFromProfile", func() {
	It("applies MinVersion and CipherSuites for a non-TLS-1.3 profile (Intermediate)", func() {
		fn, unsupported := NewTLSConfigFromProfile(*TLSProfiles[TLSProfileIntermediateType])
		Expect(unsupported).To(BeEmpty())

		conf := &tls.Config{}
		fn(conf)
		Expect(conf.MinVersion).To(Equal(uint16(tls.VersionTLS12)))
		Expect(conf.CipherSuites).NotTo(BeEmpty())
	})

	It("applies MinVersion for the Old profile and resolves all its ciphers", func() {
		fn, unsupported := NewTLSConfigFromProfile(*TLSProfiles[TLSProfileOldType])
		Expect(unsupported).To(BeEmpty())

		conf := &tls.Config{}
		fn(conf)
		Expect(conf.MinVersion).To(Equal(uint16(tls.VersionTLS10)))
		Expect(conf.CipherSuites).NotTo(BeEmpty())
	})

	It("leaves CipherSuites untouched for a TLS-1.3 profile (Modern)", func() {
		fn, _ := NewTLSConfigFromProfile(*TLSProfiles[TLSProfileModernType])

		sentinel := []uint16{tls.TLS_RSA_WITH_RC4_128_SHA}
		conf := &tls.Config{CipherSuites: sentinel}
		fn(conf)

		Expect(conf.MinVersion).To(Equal(uint16(tls.VersionTLS13)))
		// Per the documented contract: TLS 1.3 cipher suites are not
		// configurable in Go, so the pre-populated CipherSuites slice
		// must remain untouched.
		Expect(conf.CipherSuites).To(Equal(sentinel))
	})

	It("partitions supported and unsupported ciphers in input order", func() {
		profile := TLSProfileSpec{
			Ciphers: []string{
				"ECDHE-RSA-AES128-GCM-SHA256", // supported (OpenSSL)
				"BOGUS-CIPHER",                // unsupported
				"TLS_AES_128_GCM_SHA256",      // supported (IANA)
				"ALSO-BOGUS",                  // unsupported
			},
			MinTLSVersion: VersionTLS12,
		}
		fn, unsupported := NewTLSConfigFromProfile(profile)
		Expect(unsupported).To(Equal([]string{"BOGUS-CIPHER", "ALSO-BOGUS"}))

		conf := &tls.Config{}
		fn(conf)
		Expect(conf.CipherSuites).To(HaveLen(2))
		Expect(conf.CipherSuites[0]).To(Equal(uint16(tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256)))
		Expect(conf.CipherSuites[1]).To(Equal(uint16(tls.TLS_AES_128_GCM_SHA256)))
	})

	It("handles an empty cipher list", func() {
		profile := TLSProfileSpec{Ciphers: []string{}, MinTLSVersion: VersionTLS12}
		fn, unsupported := NewTLSConfigFromProfile(profile)
		Expect(unsupported).To(BeEmpty())

		conf := &tls.Config{}
		fn(conf)
		Expect(conf.MinVersion).To(Equal(uint16(tls.VersionTLS12)))
		Expect(conf.CipherSuites).To(BeEmpty())
	})
})

var _ = Describe("TLSProfiles map", func() {
	// These pinned assertions intentionally catch any change to the canned
	// Mozilla-derived profiles. If the project deliberately rebases the
	// profiles against a newer Mozilla guideline, these expectations must
	// be updated in the same change.

	It("pins the Old profile to its exact cipher list and min version", func() {
		Expect(TLSProfiles[TLSProfileOldType]).NotTo(BeNil())
		Expect(TLSProfiles[TLSProfileOldType].MinTLSVersion).To(Equal(VersionTLS10))
		Expect(TLSProfiles[TLSProfileOldType].Ciphers).To(Equal([]string{
			"TLS_AES_128_GCM_SHA256",
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
			"ECDHE-ECDSA-AES128-GCM-SHA256",
			"ECDHE-RSA-AES128-GCM-SHA256",
			"ECDHE-ECDSA-AES256-GCM-SHA384",
			"ECDHE-RSA-AES256-GCM-SHA384",
			"ECDHE-ECDSA-CHACHA20-POLY1305",
			"ECDHE-RSA-CHACHA20-POLY1305",
			"ECDHE-ECDSA-AES128-SHA256",
			"ECDHE-RSA-AES128-SHA256",
			"ECDHE-ECDSA-AES128-SHA",
			"ECDHE-RSA-AES128-SHA",
			"ECDHE-ECDSA-AES256-SHA",
			"ECDHE-RSA-AES256-SHA",
			"AES128-GCM-SHA256",
			"AES256-GCM-SHA384",
			"AES128-SHA256",
			"AES128-SHA",
			"AES256-SHA",
			"DES-CBC3-SHA",
		}))
	})

	It("pins the Intermediate profile to its exact cipher list and min version", func() {
		Expect(TLSProfiles[TLSProfileIntermediateType]).NotTo(BeNil())
		Expect(TLSProfiles[TLSProfileIntermediateType].MinTLSVersion).To(Equal(VersionTLS12))
		Expect(TLSProfiles[TLSProfileIntermediateType].Ciphers).To(Equal([]string{
			"TLS_AES_128_GCM_SHA256",
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
			"ECDHE-ECDSA-AES128-GCM-SHA256",
			"ECDHE-RSA-AES128-GCM-SHA256",
			"ECDHE-ECDSA-AES256-GCM-SHA384",
			"ECDHE-RSA-AES256-GCM-SHA384",
			"ECDHE-ECDSA-CHACHA20-POLY1305",
			"ECDHE-RSA-CHACHA20-POLY1305",
		}))
	})

	It("pins the Modern profile to its exact cipher list and min version", func() {
		Expect(TLSProfiles[TLSProfileModernType]).NotTo(BeNil())
		Expect(TLSProfiles[TLSProfileModernType].MinTLSVersion).To(Equal(VersionTLS13))
		Expect(TLSProfiles[TLSProfileModernType].Ciphers).To(Equal([]string{
			"TLS_AES_128_GCM_SHA256",
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
		}))
	})

	DescribeTable("every cipher in every canned profile resolves to a non-zero TLS code",
		func(profileType TLSProfileType) {
			profile := TLSProfiles[profileType]
			Expect(profile).NotTo(BeNil())
			for _, cipher := range profile.Ciphers {
				Expect(cipherCode(cipher)).NotTo(BeZero(), "cipher %q in profile %q must resolve", cipher, profileType)
			}
		},
		Entry("Old", TLSProfileOldType),
		Entry("Intermediate", TLSProfileIntermediateType),
		Entry("Modern", TLSProfileModernType),
	)
})
