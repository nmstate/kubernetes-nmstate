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

var _ = Describe("curveCodes", func() {
	It("maps IANA names to tls.CurveID", func() {
		codes, unsupported := curveCodes([]string{"X25519MLKEM768", "X25519", "secp256r1", "secp384r1", "secp521r1"})
		Expect(unsupported).To(BeEmpty())
		Expect(codes).To(Equal([]tls.CurveID{
			tls.X25519MLKEM768, tls.X25519, tls.CurveP256, tls.CurveP384, tls.CurveP521,
		}))
	})

	It("maps common aliases", func() {
		codes, unsupported := curveCodes([]string{"x25519", "prime256v1", "P-256", "P-384", "P-521"})
		Expect(unsupported).To(BeEmpty())
		Expect(codes).To(Equal([]tls.CurveID{
			tls.X25519, tls.CurveP256, tls.CurveP256, tls.CurveP384, tls.CurveP521,
		}))
	})

	It("reports unsupported names and keeps supported ones", func() {
		codes, unsupported := curveCodes([]string{"X25519", "brainpoolP256r1", "ffdhe2048"})
		Expect(codes).To(Equal([]tls.CurveID{tls.X25519}))
		Expect(unsupported).To(Equal([]string{"brainpoolP256r1", "ffdhe2048"}))
	})

	It("returns empty results for empty input", func() {
		codes, unsupported := curveCodes(nil)
		Expect(codes).To(BeEmpty())
		Expect(unsupported).To(BeEmpty())
	})
})
