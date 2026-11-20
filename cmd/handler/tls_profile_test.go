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

package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/nmstate/kubernetes-nmstate/pkg/cluster"
	nmstatetls "github.com/nmstate/kubernetes-nmstate/pkg/tls"
)

// The webhook and metrics pods load the TLS profile mounted from the
// operator-managed ConfigMap once at startup, and the operator restarts them
// through the pod template hash annotation after delivering a new profile.
var _ = Describe("TLS profile of the serving components", func() {
	const (
		host        = "127.0.0.1"
		dialTimeout = time.Second
		// The certificate watcher falls back to polling every 10 seconds.
		timeout  = 30 * time.Second
		interval = 100 * time.Millisecond
	)
	var (
		certDir     string
		profilePath string
		roots       *x509.CertPool
	)

	// writeServingCertificate stores a self-signed serving certificate with
	// the given serial number the way kubelet projects a Secret generation.
	writeServingCertificate := func(key *ecdsa.PrivateKey, serial int64) {
		GinkgoHelper()
		template := &x509.Certificate{
			SerialNumber:          big.NewInt(serial),
			Subject:               pkix.Name{CommonName: host},
			IPAddresses:           []net.IP{net.ParseIP(host)},
			NotBefore:             time.Now().Add(-time.Minute),
			NotAfter:              time.Now().Add(time.Hour),
			IsCA:                  true,
			BasicConstraintsValid: true,
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}
		certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		Expect(err).ToNot(HaveOccurred())
		cert, err := x509.ParseCertificate(certDER)
		Expect(err).ToNot(HaveOccurred())
		roots.AddCert(cert)
		keyDER, err := x509.MarshalECPrivateKey(key)
		Expect(err).ToNot(HaveOccurred())

		generationDir := filepath.Join(certDir, fmt.Sprintf("generation-%d", serial))
		Expect(os.Mkdir(generationDir, 0o700)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(generationDir, "tls.crt"),
			pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0o600)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(generationDir, "tls.key"),
			pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600)).To(Succeed())
	}

	// projectServingCertificate atomically switches the mounted certificate
	// to the generation with the given serial number.
	projectServingCertificate := func(serial int64) {
		GinkgoHelper()
		next := filepath.Join(certDir, "..data_tmp")
		Expect(os.Symlink(fmt.Sprintf("generation-%d", serial), next)).To(Succeed())
		Expect(os.Rename(next, filepath.Join(certDir, "..data"))).To(Succeed())
	}

	// deliverProfile atomically replaces the mounted TLS profile with the
	// content the operator writes into the ConfigMap.
	deliverProfile := func(profileType nmstatetls.TLSProfileType) {
		GinkgoHelper()
		data, err := json.Marshal(nmstatetls.TLSProfiles[profileType])
		Expect(err).ToNot(HaveOccurred())
		Expect(os.WriteFile(profilePath+".tmp", data, 0o600)).To(Succeed())
		Expect(os.Rename(profilePath+".tmp", profilePath)).To(Succeed())
	}

	freePort := func() int {
		GinkgoHelper()
		listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
		Expect(err).ToNot(HaveOccurred())
		port := listener.Addr().(*net.TCPAddr).Port
		Expect(listener.Close()).To(Succeed())
		return port
	}

	type startServer func(port int, tlsOpts func(*tls.Config)) (func(context.Context) error, error)

	startWebhookServer := func(port int, tlsOpts func(*tls.Config)) (func(context.Context) error, error) {
		server := webhook.NewServer(webhook.Options{
			Host:    host,
			Port:    port,
			CertDir: certDir,
			TLSOpts: []func(*tls.Config){tlsOpts},
		})
		return server.Start, nil
	}

	startMetricsServer := func(port int, tlsOpts func(*tls.Config)) (func(context.Context) error, error) {
		server, err := metricsserver.NewServer(metricsserver.Options{
			BindAddress:   net.JoinHostPort(host, strconv.Itoa(port)),
			SecureServing: true,
			CertDir:       certDir,
			TLSOpts:       []func(*tls.Config){tlsOpts},
		}, nil, nil)
		if err != nil {
			return nil, err
		}
		return server.Start, nil
	}

	// run starts the component the way the handler binary does at startup:
	// with the TLS options loaded from the mounted profile. It returns the
	// serving address and a function stopping the component.
	run := func(start startServer) (string, func()) {
		GinkgoHelper()
		platformTLSOpts, _, err := cluster.FetchTLSProfileFromFile(profilePath)
		Expect(err).ToNot(HaveOccurred())
		port := freePort()
		serve, err := start(port, composeTLSOpts(platformTLSOpts))
		Expect(err).ToNot(HaveOccurred())

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			defer GinkgoRecover()
			done <- serve(ctx)
		}()
		stopped := false
		stop := func() {
			GinkgoHelper()
			if stopped {
				return
			}
			stopped = true
			cancel()
			Eventually(done, timeout).Should(Receive(BeNil()))
		}
		DeferCleanup(stop)
		return net.JoinHostPort(host, strconv.Itoa(port)), stop
	}

	// servedCertificateSerial opens a new connection restricted to the given
	// TLS version and returns the serial number of the presented certificate.
	servedCertificateSerial := func(address string, version uint16) (int64, error) {
		//nolint:gosec // the offered TLS version is the subject of the test
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: dialTimeout}, "tcp", address, &tls.Config{
			RootCAs:    roots,
			MinVersion: version,
			MaxVersion: version,
		})
		if err != nil {
			return 0, err
		}
		defer conn.Close()
		return conn.ConnectionState().PeerCertificates[0].SerialNumber.Int64(), nil
	}

	BeforeEach(func() {
		certDir = GinkgoT().TempDir()
		profilePath = filepath.Join(GinkgoT().TempDir(), "profile.json")
		roots = x509.NewCertPool()

		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		Expect(err).ToNot(HaveOccurred())
		writeServingCertificate(key, 1)
		writeServingCertificate(key, 2)
		projectServingCertificate(1)
		for _, name := range []string{"tls.crt", "tls.key"} {
			Expect(os.Symlink(filepath.Join("..data", name), filepath.Join(certDir, name))).To(Succeed())
		}
	})

	DescribeTable("should reject TLS 1.2 on new connections once restarted on a delivered Modern profile",
		func(start startServer) {
			deliverProfile(nmstatetls.TLSProfileIntermediateType)
			address, stop := run(start)
			Eventually(servedCertificateSerial).WithArguments(address, uint16(tls.VersionTLS12)).
				WithTimeout(timeout).WithPolling(interval).Should(Equal(int64(1)))

			By("delivering the Modern profile and rotating the serving certificate")
			deliverProfile(nmstatetls.TLSProfileModernType)
			projectServingCertificate(2)

			By("hot reloading the serving certificate without a restart")
			Eventually(servedCertificateSerial).WithArguments(address, uint16(tls.VersionTLS12)).
				WithTimeout(timeout).WithPolling(interval).Should(Equal(int64(2)))

			By("restarting the component as the changed pod template does")
			stop()
			address, _ = run(start)
			Eventually(servedCertificateSerial).WithArguments(address, uint16(tls.VersionTLS13)).
				WithTimeout(timeout).WithPolling(interval).Should(Equal(int64(2)))
			_, err := servedCertificateSerial(address, tls.VersionTLS12)
			Expect(err).To(MatchError(ContainSubstring("protocol version not supported")))
		},
		Entry("webhook", startWebhookServer),
		Entry("metrics", startMetricsServer),
	)
})
