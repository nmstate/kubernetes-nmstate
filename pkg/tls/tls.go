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
