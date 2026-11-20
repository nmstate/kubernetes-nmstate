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
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Nginx TLS profile conversion", func() {
	It("deduplicates group aliases without changing the first preference order", func() {
		got, unsupported, err := NewNginxConfigFromProfile(TLSProfileSpec{
			MinTLSVersion: VersionTLS13,
			Curves:        []string{"P-256", "X25519", "secp256r1", "prime256v1", "x25519", "P-384", "secp384r1"},
		})
		if err != nil || !unsupported.IsEmpty() {
			GinkgoT().Fatalf("unexpected conversion error: %v, unsupported=%#v", err, unsupported)
		}
		if got.Curves != "prime256v1:X25519:secp384r1" {
			GinkgoT().Fatalf("duplicate canonical groups or changed order: %s", got.Curves)
		}
	})
	It("translates supported profiles and protocol families", func() { testNginxTLSProfiles(GinkgoTB()) })
	It("keeps legacy and modern cipher families separate", func() { testNginxOldTLSProfile(GinkgoTB()) })
	It("rejects unrepresentable and injection-bearing profiles", func() { testNginxRejectsUnrepresentableProfiles(GinkgoTB()) })
	It("reports unsupported entries while retaining a usable subset", func() { testNginxPartiallySupportedProfile(GinkgoTB()) })
	It("reports groups unavailable in the plugin OpenSSL", func() { testNginxReportsGroupsUnavailableInPluginOpenSSL(GinkgoTB()) })
})

func testNginxTLSProfiles(t testing.TB) {
	tests := []struct {
		name    string
		profile TLSProfileSpec
		want    NginxConfig
	}{
		{
			name:    "modern",
			profile: *TLSProfiles[TLSProfileModernType],
			want: NginxConfig{
				Protocols:    "TLSv1.3",
				CipherSuites: "TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256",
			},
		},
		{
			name:    "intermediate",
			profile: *TLSProfiles[TLSProfileIntermediateType],
			want: NginxConfig{
				Protocols: "TLSv1.2 TLSv1.3",
				Ciphers: "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:" +
					"ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:" +
					"ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305",
				CipherSuites: "TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256",
			},
		},
		{
			name: "custom IANA ciphers and curve aliases",
			profile: TLSProfileSpec{
				MinTLSVersion: VersionTLS12,
				Ciphers:       []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"},
				Curves:        []string{"P-256", "x25519"},
			},
			want: NginxConfig{
				Protocols: "TLSv1.2 TLSv1.3", Ciphers: "ECDHE-RSA-AES128-GCM-SHA256",
				CipherSuites: "TLS_AES_256_GCM_SHA384", Curves: "prime256v1:X25519",
			},
		},
		{
			name:    "explicit TLS12 ciphers exclude TLS13",
			profile: TLSProfileSpec{MinTLSVersion: VersionTLS12, Ciphers: []string{"ECDHE-RSA-AES256-GCM-SHA384"}},
			want:    NginxConfig{Protocols: "TLSv1.2", Ciphers: "ECDHE-RSA-AES256-GCM-SHA384"},
		},
		{
			name:    "only TLS13 ciphers raise effective minimum",
			profile: TLSProfileSpec{MinTLSVersion: VersionTLS10, Ciphers: []string{"TLS_AES_128_GCM_SHA256"}},
			want:    NginxConfig{Protocols: "TLSv1.3", CipherSuites: "TLS_AES_128_GCM_SHA256"},
		},
		{
			name:    "empty cipher list leaves crypto defaults",
			profile: TLSProfileSpec{MinTLSVersion: VersionTLS11},
			want:    NginxConfig{Protocols: "TLSv1.1 TLSv1.2 TLSv1.3"},
		},
		{
			name:    "unspecified minimum matches existing converter default",
			profile: TLSProfileSpec{},
			want:    NginxConfig{Protocols: "TLSv1.2 TLSv1.3"},
		},
	}
	for _, tt := range tests {
		By(tt.name)
		got, unsupported, err := NewNginxConfigFromProfile(tt.profile)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Fatalf("got %#v; want %#v", got, tt.want)
		}
		if !unsupported.IsEmpty() {
			t.Fatalf("fully supported profile reported unsupported values: %#v", unsupported)
		}
	}
}

func testNginxOldTLSProfile(t testing.TB) {
	got, _, err := NewNginxConfigFromProfile(*TLSProfiles[TLSProfileOldType])
	if err != nil {
		t.Fatal(err)
	}
	if got.Protocols != "TLSv1 TLSv1.1 TLSv1.2 TLSv1.3" ||
		!strings.Contains(got.Ciphers, "AES128-SHA") ||
		strings.Contains(got.Ciphers, "TLS_AES_") {
		t.Fatalf("old profile not correctly separated into protocol/cipher families: %#v", got)
	}
}

func testNginxRejectsUnrepresentableProfiles(t testing.TB) {
	for name, profile := range map[string]TLSProfileSpec{
		"unknown version":             {MinTLSVersion: "future"},
		"unknown cipher":              {Ciphers: []string{"not-a-cipher"}},
		"unknown curve":               {Curves: []string{"not-a-curve"}},
		"cipher injection":            {Ciphers: []string{"ECDHE-RSA-AES128-GCM-SHA256;\nserver {}"}},
		"curve injection":             {Curves: []string{"X25519; ssl_protocols TLSv1;"}},
		"empty protocol intersection": {MinTLSVersion: VersionTLS13, Ciphers: []string{"ECDHE-RSA-AES128-GCM-SHA256"}},
	} {
		By(name)
		if _, _, err := NewNginxConfigFromProfile(profile); err == nil {
			t.Fatal("expected an error rather than a broader or invalid nginx configuration")
		}
	}
}

func testNginxPartiallySupportedProfile(t testing.TB) {
	profile := TLSProfileSpec{
		MinTLSVersion: VersionTLS12,
		Ciphers:       []string{"ECDHE-RSA-AES128-GCM-SHA256", "DHE-RSA-AES128-GCM-SHA256"},
		Curves:        []string{"X25519", "unknown-group"},
	}
	got, unsupported, err := NewNginxConfigFromProfile(profile)
	if err != nil {
		t.Fatalf("a partially supported profile should remain usable, as it does for the Go operands: %v", err)
	}
	if got.Ciphers != "ECDHE-RSA-AES128-GCM-SHA256" || got.Curves != "X25519" {
		t.Fatalf("unexpected supported subset: %#v", got)
	}
	if len(unsupported.Ciphers) != 1 || unsupported.Ciphers[0] != "DHE-RSA-AES128-GCM-SHA256" ||
		len(unsupported.Curves) != 1 || unsupported.Curves[0] != "unknown-group" {
		t.Fatalf("unsupported values not reported: %#v", unsupported)
	}
}

func testNginxReportsGroupsUnavailableInPluginOpenSSL(t testing.TB) {
	got, unsupported, err := NewNginxConfigFromProfile(TLSProfileSpec{
		MinTLSVersion: VersionTLS13,
		Curves:        []string{"X25519", "X25519MLKEM768"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Curves != "X25519" || len(unsupported.Curves) != 1 || unsupported.Curves[0] != "X25519MLKEM768" {
		t.Fatalf("plugin image OpenSSL 3.0 does not support the hybrid group: config=%#v unsupported=%#v", got, unsupported)
	}
}
