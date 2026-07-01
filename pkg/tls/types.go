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

// TLSProfileType defines a TLS security profile type.
type TLSProfileType string

const (
	TLSProfileOldType          TLSProfileType = "Old"
	TLSProfileIntermediateType TLSProfileType = "Intermediate"
	TLSProfileModernType       TLSProfileType = "Modern"
	TLSProfileCustomType       TLSProfileType = "Custom"
)

// TLSProtocolVersion is a way to specify the protocol version used for TLS connections.
type TLSProtocolVersion string

const (
	VersionTLS10 TLSProtocolVersion = "VersionTLS10"
	VersionTLS11 TLSProtocolVersion = "VersionTLS11"
	VersionTLS12 TLSProtocolVersion = "VersionTLS12"
	VersionTLS13 TLSProtocolVersion = "VersionTLS13"
)

// TLSProfileSpec is the desired behavior of a TLSSecurityProfile.
type TLSProfileSpec struct {
	Ciphers       []string           `json:"ciphers"`
	MinTLSVersion TLSProtocolVersion `json:"minTLSVersion"`
}

// TLSProfiles contains the predefined TLS profile specifications.
// Based on version 5.7 of the Mozilla Server Side TLS configuration guidelines.
// See: https://ssl-config.mozilla.org/guidelines/5.7.json
var TLSProfiles = map[TLSProfileType]*TLSProfileSpec{
	TLSProfileOldType: {
		Ciphers: []string{
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
		},
		MinTLSVersion: VersionTLS10,
	},
	TLSProfileIntermediateType: {
		Ciphers: []string{
			"TLS_AES_128_GCM_SHA256",
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
			"ECDHE-ECDSA-AES128-GCM-SHA256",
			"ECDHE-RSA-AES128-GCM-SHA256",
			"ECDHE-ECDSA-AES256-GCM-SHA384",
			"ECDHE-RSA-AES256-GCM-SHA384",
			"ECDHE-ECDSA-CHACHA20-POLY1305",
			"ECDHE-RSA-CHACHA20-POLY1305",
		},
		MinTLSVersion: VersionTLS12,
	},
	TLSProfileModernType: {
		Ciphers: []string{
			"TLS_AES_128_GCM_SHA256",
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
		},
		MinTLSVersion: VersionTLS13,
	},
}
