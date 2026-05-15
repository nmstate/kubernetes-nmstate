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

	apiServerGVK = schema.GroupVersionKind{
		Group:   "config.openshift.io",
		Version: "v1",
		Kind:    "APIServer",
	}
)

// APIServerTLSConfig bundles the TLS profile spec and the cluster-wide
// TLS adherence policy as observed from the cluster's APIServer resource.
type APIServerTLSConfig struct {
	Profile   TLSProfileSpec
	Adherence TLSAdherencePolicy
}

// FetchAPIServerTLSConfig fetches both the TLS profile spec and the
// tlsAdherence policy from the cluster's APIServer resource. A missing
// or empty tlsAdherence field is returned as the zero value ("").
func FetchAPIServerTLSConfig(ctx context.Context, k8sClient client.Client) (APIServerTLSConfig, error) {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(apiServerGVK)

	key := client.ObjectKey{Name: apiServerName}
	if err := k8sClient.Get(ctx, key, obj); err != nil {
		return APIServerTLSConfig{}, fmt.Errorf("failed to get APIServer %q: %w", key.String(), err)
	}

	profile, err := parseTLSSecurityProfile(obj.Object)
	if err != nil {
		return APIServerTLSConfig{}, fmt.Errorf("failed to parse TLS profile from APIServer %q: %w", key.String(), err)
	}

	spec, err := GetTLSProfileSpec(profile)
	if err != nil {
		return APIServerTLSConfig{}, fmt.Errorf("failed to get TLS profile from APIServer %q: %w", key.String(), err)
	}

	adherence := parseTLSAdherence(obj.Object)

	return APIServerTLSConfig{Profile: spec, Adherence: adherence}, nil
}

// ShouldHonorClusterTLSProfile returns true when the caller must honor the
// cluster-wide TLS profile. It mirrors the semantics of the library-go helper
// of the same name (openshift/library-go #2114): "" and
// LegacyAdheringComponentsOnly return false; any other value (including
// StrictAllComponents and any unknown future value) returns true for
// forward compatibility.
//
// kubernetes-nmstate is already a legacy-adhering component: it honors the
// cluster TLS profile in all observed adherence states. This helper exists so
// any future code path that needs to gate behavior on adherence has a single,
// consistent definition to call.
func ShouldHonorClusterTLSProfile(adherence TLSAdherencePolicy) bool {
	switch adherence {
	case "", TLSAdherenceLegacyAdheringComponentsOnly:
		return false
	case TLSAdherenceStrictAllComponents:
		return true
	default:
		// Any unknown future value: assume strict for forward compatibility.
		return true
	}
}

// parseTLSSecurityProfile extracts the TLS profile type and custom spec from an unstructured APIServer object.
func parseTLSSecurityProfile(obj map[string]interface{}) (*tlsSecurityProfile, error) {
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

		profile.customSpec = &TLSProfileSpec{
			Ciphers:       ciphersRaw,
			MinTLSVersion: TLSProtocolVersion(minVersion),
		}
	}

	return profile, nil
}

// tlsSecurityProfile holds the parsed TLS profile data from the APIServer resource.
type tlsSecurityProfile struct {
	profileType TLSProfileType
	customSpec  *TLSProfileSpec
}

// parseTLSAdherence extracts spec.tlsAdherence from an unstructured APIServer
// object. A missing field, non-string value, or read error is treated as
// "" (no opinion) — this keeps callers safe on clusters where the feature
// gate is disabled or the field is not yet present.
func parseTLSAdherence(obj map[string]interface{}) TLSAdherencePolicy {
	value, found, err := unstructured.NestedString(obj, "spec", "tlsAdherence")
	if err != nil || !found {
		return ""
	}
	return TLSAdherencePolicy(value)
}

// GetTLSProfileSpec returns a TLSProfileSpec for the given profile.
// If no profile is configured, the default Intermediate profile is returned.
func GetTLSProfileSpec(profile *tlsSecurityProfile) (TLSProfileSpec, error) {
	defaultProfile := *TLSProfiles[TLSProfileIntermediateType]

	if profile == nil || profile.profileType == "" {
		return defaultProfile, nil
	}

	if profile.profileType != TLSProfileCustomType {
		if tlsConfig, ok := TLSProfiles[profile.profileType]; ok {
			return *tlsConfig, nil
		}
		return defaultProfile, nil
	}

	if profile.customSpec == nil {
		return TLSProfileSpec{}, ErrCustomProfileNil
	}

	return *profile.customSpec, nil
}

// NewTLSConfigFromProfile returns a function that configures a tls.Config based on the provided TLSProfileSpec,
// along with any cipher names from the profile that are not supported.
func NewTLSConfigFromProfile(profile TLSProfileSpec) (tlsConfig func(*tls.Config), unsupportedCiphers []string) {
	minVersion := tlsVersionOrDie(string(profile.MinTLSVersion))
	cipherSuites, unsupported := cipherCodes(profile.Ciphers)

	return func(tlsConf *tls.Config) {
		tlsConf.MinVersion = minVersion
		// TLS 1.3 cipher suites are not configurable in Go.
		if minVersion != tls.VersionTLS13 {
			tlsConf.CipherSuites = cipherSuites
		}
	}, unsupported
}
