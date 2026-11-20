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
	"fmt"
	"slices"
	"strings"
)

// NginxConfig holds directive values for an nginx TLS server.
type NginxConfig struct {
	Protocols    string
	Ciphers      string
	CipherSuites string
	Curves       string
}

// NewNginxConfigFromProfile translates the cluster profile into nginx directives.
// Cipher names must be individual suites, not OpenSSL expressions. Only known
// names are emitted so profile values cannot introduce nginx directives.
// As with the Go converter, unsupported entries are reported and a usable
// subset is retained; an empty usable protocol or group set is an error.
func NewNginxConfigFromProfile(profile TLSProfileSpec) (NginxConfig, UnsupportedEntries, error) {
	var unsupported UnsupportedEntries
	minVersion, err := tlsVersion(string(profile.MinTLSVersion))
	if err != nil {
		return NginxConfig{}, unsupported, err
	}

	var legacyCiphers, modernCiphers, curves []string
	for _, cipher := range profile.Ciphers {
		name, err := nginxCipherName(cipher)
		if err != nil {
			unsupported.Ciphers = append(unsupported.Ciphers, cipher)
			continue
		}
		if tls13CipherSuiteIDs[cipherCode(cipher)] {
			modernCiphers = append(modernCiphers, name)
		} else {
			legacyCiphers = append(legacyCiphers, name)
		}
	}
	for _, curve := range profile.Curves {
		name, ok := nginxCurves[curveIDs[curve]]
		if !ok {
			unsupported.Curves = append(unsupported.Curves, curve)
			continue
		}
		if !slices.Contains(curves, name) {
			curves = append(curves, name)
		}
	}
	if len(profile.Curves) > 0 && len(curves) == 0 {
		return NginxConfig{}, unsupported, ErrNoSupportedCurves
	}

	var protocols []string
	for _, protocol := range []struct {
		version uint16
		name    string
	}{
		{tls.VersionTLS10, "TLSv1"},
		{tls.VersionTLS11, "TLSv1.1"},
		{tls.VersionTLS12, "TLSv1.2"},
		{tls.VersionTLS13, "TLSv1.3"},
	} {
		if protocol.version < minVersion {
			continue
		}
		// An explicit cipher allowlist must not accidentally enable a
		// protocol family with its unrestricted default cipher suites.
		if len(profile.Ciphers) > 0 &&
			((protocol.version == tls.VersionTLS13 && len(modernCiphers) == 0) ||
				(protocol.version < tls.VersionTLS13 && len(legacyCiphers) == 0)) {
			continue
		}
		protocols = append(protocols, protocol.name)
	}
	if len(protocols) == 0 {
		return NginxConfig{}, unsupported,
			fmt.Errorf("TLS profile has no nginx ciphers compatible with minimum version %s", profile.MinTLSVersion)
	}
	if minVersion == tls.VersionTLS13 {
		legacyCiphers = nil
	}
	return NginxConfig{
		Protocols:    strings.Join(protocols, " "),
		Ciphers:      strings.Join(legacyCiphers, ":"),
		CipherSuites: strings.Join(modernCiphers, ":"),
		Curves:       strings.Join(curves, ":"),
	}, unsupported, nil
}

// The default plugin image uses OpenSSL 3.0, which cannot configure the
// X25519MLKEM768 group supported by Go. Report it as unsupported rather than
// render a directive that prevents nginx from starting.
var nginxCurves = map[tls.CurveID]string{
	tls.X25519:    "X25519",
	tls.CurveP256: "prime256v1",
	tls.CurveP384: "secp384r1",
	tls.CurveP521: "secp521r1",
}

func nginxCipherName(cipher string) (string, error) {
	if _, ok := openSSLToIANA[cipher]; ok {
		return cipher, nil
	}
	if code := cipherCode(cipher); code != 0 {
		for openssl, iana := range openSSLToIANA {
			if ianaCiphers[iana] == code {
				return openssl, nil
			}
		}
	}
	return "", fmt.Errorf("unsupported nginx TLS cipher %q", cipher)
}
