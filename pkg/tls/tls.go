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
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	apiServerName = "cluster"
)

var (
	// ErrCustomProfileNil is returned when a custom TLS profile is specified but the Custom field is nil.
	ErrCustomProfileNil = errors.New("custom TLS profile specified but Custom field is nil")

	// ErrUnknownProfileType is returned when the profile type is not recognized.
	ErrUnknownProfileType = errors.New("unknown TLS profile type")

	// ErrNoSupportedCiphers is returned when none of the profile ciphers is supported.
	ErrNoSupportedCiphers = errors.New("TLS profile specifies ciphers but none are supported")

	// ErrNoSupportedCurves is returned when none of the profile curves is supported.
	ErrNoSupportedCurves = errors.New("TLS profile specifies curves but none are supported")

	apiServerGVK = schema.GroupVersionKind{
		Group:   "config.openshift.io",
		Version: "v1",
		Kind:    "APIServer",
	}
)

// FetchAPIServerTLSProfile fetches the TLS profile spec from the cluster's APIServer resource.
func FetchAPIServerTLSProfile(ctx context.Context, k8sClient client.Client) (TLSProfileSpec, error) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(apiServerGVK)

	key := client.ObjectKey{Name: apiServerName}
	if err := k8sClient.Get(ctx, key, obj); err != nil {
		return TLSProfileSpec{}, fmt.Errorf("failed to get APIServer %q: %w", key.String(), err)
	}

	profile, err := parseTLSSecurityProfile(obj.Object)
	if err != nil {
		return TLSProfileSpec{}, fmt.Errorf("failed to parse TLS profile from APIServer %q: %w", key.String(), err)
	}

	spec, err := GetTLSProfileSpec(profile)
	if err != nil {
		return TLSProfileSpec{}, fmt.Errorf("failed to get TLS profile from APIServer %q: %w", key.String(), err)
	}

	return spec, nil
}

// parseTLSSecurityProfile extracts the TLS profile type and custom spec from an unstructured APIServer object.
func parseTLSSecurityProfile(obj map[string]any) (*tlsSecurityProfile, error) {
	profileMap, found, err := unstructured.NestedMap(obj, "spec", "tlsSecurityProfile")
	if err != nil {
		return nil, fmt.Errorf("failed to extract spec.tlsSecurityProfile: %w", err)
	}
	if !found || profileMap == nil {
		return nil, nil
	}

	profileType, _, _ := unstructured.NestedString(profileMap, "type")

	profile := &tlsSecurityProfile{
		profileType: TLSProfileType(profileType),
	}

	if profile.profileType == TLSProfileCustomType {
		customMap, found, _ := unstructured.NestedMap(profileMap, "custom")
		if !found || customMap == nil {
			return profile, nil
		}

		minVersion, _, _ := unstructured.NestedString(customMap, "minTLSVersion")

		ciphersRaw, _, _ := unstructured.NestedStringSlice(customMap, "ciphers")

		curvesRaw, _, _ := unstructured.NestedStringSlice(customMap, "curves")

		profile.customSpec = &TLSProfileSpec{
			Ciphers:       ciphersRaw,
			MinTLSVersion: TLSProtocolVersion(minVersion),
			Curves:        curvesRaw,
		}
	}

	return profile, nil
}

// tlsSecurityProfile holds the parsed TLS profile data from the APIServer resource.
type tlsSecurityProfile struct {
	profileType TLSProfileType
	customSpec  *TLSProfileSpec
}

// GetTLSProfileSpec returns a TLSProfileSpec for the given profile.
// If no profile is configured, the default Intermediate profile is returned;
// unknown profile types are an error.
func GetTLSProfileSpec(profile *tlsSecurityProfile) (TLSProfileSpec, error) {
	if profile == nil || profile.profileType == "" {
		return *TLSProfiles[TLSProfileIntermediateType], nil
	}

	if profile.profileType != TLSProfileCustomType {
		if tlsConfig, ok := TLSProfiles[profile.profileType]; ok {
			return *tlsConfig, nil
		}
		return TLSProfileSpec{}, fmt.Errorf("%w: %q", ErrUnknownProfileType, profile.profileType)
	}

	if profile.customSpec == nil {
		return TLSProfileSpec{}, ErrCustomProfileNil
	}

	return *profile.customSpec, nil
}

// tls13CipherSuiteNames are the TLS 1.3 cipher suites Go always enables;
// they cannot be restricted via tls.Config.CipherSuites.
var tls13CipherSuiteNames = []string{
	"TLS_AES_128_GCM_SHA256",
	"TLS_AES_256_GCM_SHA384",
	"TLS_CHACHA20_POLY1305_SHA256",
}

// UnsupportedEntries lists profile entries that were dropped because Go does
// not support them, and restrictions that Go cannot enforce.
type UnsupportedEntries struct {
	Ciphers []string
	Curves  []string
	// UnenforceableCiphers lists TLS 1.3 cipher suites that Go always
	// offers even though the profile excludes them.
	UnenforceableCiphers []string
}

// IsEmpty returns true when every profile entry is supported and enforceable.
func (u UnsupportedEntries) IsEmpty() bool {
	return len(u.Ciphers) == 0 && len(u.Curves) == 0 && len(u.UnenforceableCiphers) == 0
}

// Message returns a human-readable description of the entries.
func (u UnsupportedEntries) Message() string {
	parts := []string{}
	if len(u.Ciphers) > 0 {
		parts = append(parts, fmt.Sprintf("unsupported ciphers ignored: %s", strings.Join(u.Ciphers, ", ")))
	}
	if len(u.Curves) > 0 {
		parts = append(parts, fmt.Sprintf("unsupported curves ignored: %s", strings.Join(u.Curves, ", ")))
	}
	if len(u.UnenforceableCiphers) > 0 {
		parts = append(parts, fmt.Sprintf(
			"Go cannot restrict TLS 1.3 cipher suites, these are offered beyond the profile: %s",
			strings.Join(u.UnenforceableCiphers, ", ")))
	}
	return strings.Join(parts, "; ")
}

// unenforceableTLS13Ciphers returns the TLS 1.3 cipher suites that Go will
// offer even though the profile's cipher list excludes them.
func unenforceableTLS13Ciphers(profileCiphers []string) []string {
	if len(profileCiphers) == 0 {
		return nil
	}
	allowed := map[string]bool{}
	for _, c := range profileCiphers {
		allowed[c] = true
	}
	unenforceable := []string{}
	for _, c := range tls13CipherSuiteNames {
		if !allowed[c] {
			unenforceable = append(unenforceable, c)
		}
	}
	return unenforceable
}

// NewTLSConfigFromProfile returns a function that configures a tls.Config
// based on the provided TLSProfileSpec, along with any profile entries that
// could not be honored. It returns an error when the profile cannot be
// honored at all, instead of falling back to Go defaults that may exceed it.
func NewTLSConfigFromProfile(profile TLSProfileSpec) (func(*tls.Config), UnsupportedEntries, error) {
	minVersion, err := tlsVersion(string(profile.MinTLSVersion))
	if err != nil {
		return nil, UnsupportedEntries{}, err
	}
	cipherSuites, unsupportedCiphers := cipherCodes(profile.Ciphers)
	curves, unsupportedCurves := curveCodes(profile.Curves)
	unsupported := UnsupportedEntries{
		Ciphers:              unsupportedCiphers,
		Curves:               unsupportedCurves,
		UnenforceableCiphers: unenforceableTLS13Ciphers(profile.Ciphers),
	}

	if len(profile.Ciphers) > 0 && len(cipherSuites) == 0 && minVersion < tls.VersionTLS13 {
		return nil, unsupported, ErrNoSupportedCiphers
	}
	if len(profile.Curves) > 0 && len(curves) == 0 {
		return nil, unsupported, ErrNoSupportedCurves
	}

	// tls.Config.CipherSuites only applies to TLS 1.2 and older. When the
	// profile permits no TLS 1.2 cipher, serve TLS 1.3 only: leaving
	// CipherSuites nil would fall back to Go's default TLS 1.2 ciphers.
	tls12CipherSuites := tls12CipherCodes(cipherSuites)
	if len(profile.Ciphers) > 0 && len(tls12CipherSuites) == 0 && minVersion < tls.VersionTLS13 {
		minVersion = tls.VersionTLS13
	}

	return func(tlsConf *tls.Config) {
		tlsConf.MinVersion = minVersion
		// TLS 1.3 cipher suites are not configurable in Go.
		if minVersion != tls.VersionTLS13 {
			tlsConf.CipherSuites = tls12CipherSuites
		}
		if len(curves) > 0 {
			tlsConf.CurvePreferences = curves
		}
	}, unsupported, nil
}
