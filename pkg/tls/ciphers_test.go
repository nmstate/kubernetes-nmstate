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
	"crypto/tls"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("tlsVersion", func() {
	DescribeTable("returns the right TLS code for a known version name",
		func(name string, expected uint16) {
			v, err := tlsVersion(name)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal(expected))
		},
		Entry("TLS 1.0", "VersionTLS10", uint16(tls.VersionTLS10)),
		Entry("TLS 1.1", "VersionTLS11", uint16(tls.VersionTLS11)),
		Entry("TLS 1.2", "VersionTLS12", uint16(tls.VersionTLS12)),
		Entry("TLS 1.3", "VersionTLS13", uint16(tls.VersionTLS13)),
	)

	It("defaults to TLS 1.2 when the version name is empty", func() {
		v, err := tlsVersion("")
		Expect(err).NotTo(HaveOccurred())
		Expect(v).To(Equal(uint16(tls.VersionTLS12)))
	})

	It("returns an error for an unknown version name", func() {
		_, err := tlsVersion("VersionTLS99")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unknown tls version"))
	})
})

var _ = Describe("tlsVersionOrDie", func() {
	It("returns the resolved code for a known version", func() {
		Expect(tlsVersionOrDie("VersionTLS12")).To(Equal(uint16(tls.VersionTLS12)))
	})

	It("returns the default for an empty version name", func() {
		Expect(tlsVersionOrDie("")).To(Equal(uint16(tls.VersionTLS12)))
	})

	It("panics on an unknown version name", func() {
		Expect(func() { tlsVersionOrDie("VersionTLS99") }).To(Panic())
	})
})

var _ = Describe("cipherSuite", func() {
	It("returns the code for a known IANA cipher name", func() {
		code, err := cipherSuite("TLS_AES_128_GCM_SHA256")
		Expect(err).NotTo(HaveOccurred())
		Expect(code).To(Equal(uint16(tls.TLS_AES_128_GCM_SHA256)))
	})

	It("returns an error for an unknown cipher name", func() {
		_, err := cipherSuite("NOPE")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unknown cipher name"))
	})

	It("returns an error for an OpenSSL-style cipher name (not in IANA map)", func() {
		_, err := cipherSuite("ECDHE-RSA-AES128-GCM-SHA256")
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("openSSLToIANACipherSuites", func() {
	It("maps every known OpenSSL name to its IANA name, in order", func() {
		input := []string{
			"ECDHE-RSA-AES128-GCM-SHA256",
			"ECDHE-ECDSA-AES256-GCM-SHA384",
			"AES128-SHA",
		}
		Expect(openSSLToIANACipherSuites(input)).To(Equal([]string{
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
			"TLS_RSA_WITH_AES_128_CBC_SHA",
		}))
	})

	It("silently drops unknown OpenSSL names", func() {
		input := []string{"ECDHE-RSA-AES128-GCM-SHA256", "NOT-A-CIPHER"}
		Expect(openSSLToIANACipherSuites(input)).To(Equal([]string{
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		}))
	})

	It("returns an empty slice for an empty input", func() {
		Expect(openSSLToIANACipherSuites(nil)).To(BeEmpty())
		Expect(openSSLToIANACipherSuites([]string{})).To(BeEmpty())
	})

	It("does not pass through an IANA name directly (only OpenSSL names are in the table)", func() {
		// Sanity check: the table is OpenSSL-keyed, so feeding an IANA-style
		// name yields no output. cipherCode() compensates by trying IANA first.
		input := []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"}
		Expect(openSSLToIANACipherSuites(input)).To(BeEmpty())
	})
})

var _ = Describe("cipherCode", func() {
	It("resolves an IANA cipher name", func() {
		Expect(cipherCode("TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")).
			To(Equal(uint16(tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256)))
	})

	It("resolves an OpenSSL cipher name via the translation table", func() {
		Expect(cipherCode("ECDHE-RSA-AES128-GCM-SHA256")).
			To(Equal(uint16(tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256)))
	})

	It("resolves a TLS 1.3 cipher name (shared between IANA and OpenSSL)", func() {
		Expect(cipherCode("TLS_AES_128_GCM_SHA256")).
			To(Equal(uint16(tls.TLS_AES_128_GCM_SHA256)))
	})

	It("returns 0 for an unknown cipher name", func() {
		Expect(cipherCode("DEFINITELY-NOT-A-CIPHER")).To(BeZero())
	})

	It("returns 0 for an empty cipher name", func() {
		Expect(cipherCode("")).To(BeZero())
	})
})

var _ = Describe("cipherCodes", func() {
	It("returns codes for all-known inputs and no unsupported list", func() {
		codes, unsupported := cipherCodes([]string{
			"ECDHE-RSA-AES128-GCM-SHA256",
			"TLS_AES_128_GCM_SHA256",
		})
		Expect(unsupported).To(BeEmpty())
		Expect(codes).To(Equal([]uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
		}))
	})

	It("returns empty codes and the full input as unsupported when nothing is known", func() {
		codes, unsupported := cipherCodes([]string{"BOGUS-1", "BOGUS-2"})
		Expect(codes).To(BeEmpty())
		Expect(unsupported).To(Equal([]string{"BOGUS-1", "BOGUS-2"}))
	})

	It("partitions a mixed input while preserving input order", func() {
		codes, unsupported := cipherCodes([]string{
			"ECDHE-RSA-AES128-GCM-SHA256",
			"BOGUS-1",
			"TLS_AES_128_GCM_SHA256",
			"BOGUS-2",
		})
		Expect(codes).To(Equal([]uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_AES_128_GCM_SHA256,
		}))
		Expect(unsupported).To(Equal([]string{"BOGUS-1", "BOGUS-2"}))
	})

	It("returns empty results for an empty or nil input", func() {
		codes, unsupported := cipherCodes(nil)
		Expect(codes).To(BeEmpty())
		Expect(unsupported).To(BeEmpty())

		codes, unsupported = cipherCodes([]string{})
		Expect(codes).To(BeEmpty())
		Expect(unsupported).To(BeEmpty())
	})
})
