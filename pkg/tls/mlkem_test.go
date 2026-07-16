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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	cryptotls "crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// selfSignedCert generates an in-memory ECDSA certificate for 127.0.0.1.
func selfSignedCert() cryptotls.Certificate {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).NotTo(HaveOccurred())
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	Expect(err).NotTo(HaveOccurred())
	return cryptotls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// handshake starts a TLS server configured through NewTLSConfigFromProfile
// and connects with a default Go client. It returns the negotiated
// connection state observed by the client.
func handshake(profile TLSProfileSpec) cryptotls.ConnectionState {
	opts, _, err := NewTLSConfigFromProfile(profile)
	Expect(err).NotTo(HaveOccurred())

	serverConf := &cryptotls.Config{Certificates: []cryptotls.Certificate{selfSignedCert()}}
	opts(serverConf)

	listener, err := cryptotls.Listen("tcp", "127.0.0.1:0", serverConf)
	Expect(err).NotTo(HaveOccurred())
	defer listener.Close()

	done := make(chan struct{})
	go func() {
		defer GinkgoRecover()
		defer close(done)
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		// Drive the handshake from the server side.
		_ = conn.(*cryptotls.Conn).HandshakeContext(context.Background())
	}()

	clientConf := &cryptotls.Config{InsecureSkipVerify: true} //nolint:gosec // test-only client
	dialer := &cryptotls.Dialer{Config: clientConf}
	rawConn, err := dialer.DialContext(context.Background(), "tcp", listener.Addr().String())
	Expect(err).NotTo(HaveOccurred())
	conn := rawConn.(*cryptotls.Conn)
	defer conn.Close()
	state := conn.ConnectionState()
	<-done
	return state
}

var _ = Describe("ML-KEM key exchange", func() {
	It("is negotiated under the Intermediate profile", func() {
		state := handshake(*TLSProfiles[TLSProfileIntermediateType])
		Expect(state.Version).To(Equal(uint16(cryptotls.VersionTLS13)))
		Expect(state.CurveID).To(Equal(cryptotls.X25519MLKEM768))
	})

	It("is negotiated under the Modern profile", func() {
		state := handshake(*TLSProfiles[TLSProfileModernType])
		Expect(state.Version).To(Equal(uint16(cryptotls.VersionTLS13)))
		Expect(state.CurveID).To(Equal(cryptotls.X25519MLKEM768))
	})

	It("is negotiated when the profile explicitly allows it", func() {
		state := handshake(TLSProfileSpec{
			MinTLSVersion: VersionTLS13,
			Curves:        []string{"X25519MLKEM768", "X25519"},
		})
		Expect(state.CurveID).To(Equal(cryptotls.X25519MLKEM768))
	})

	It("is NOT negotiated when the profile restricts curves", func() {
		state := handshake(TLSProfileSpec{
			MinTLSVersion: VersionTLS13,
			Curves:        []string{"secp256r1"},
		})
		Expect(state.CurveID).To(Equal(cryptotls.CurveP256))
	})
})
