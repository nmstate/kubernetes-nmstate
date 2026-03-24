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

package cluster

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"

	configv1 "github.com/openshift/api/config/v1"
	securityv1 "github.com/openshift/api/security/v1"
	tlspkg "github.com/openshift/controller-runtime-common/pkg/tls"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("cluster")

// IsOpenShift returns true if the current cluster is an OpenShift/OKD cluster.
func IsOpenShift(kclient client.Client) (bool, error) {
	// if the cluster has the securityContextConstraint resource of the group security.openshift.io, then it is most likely an OCP/OKD cluster
	_, err := kclient.RESTMapper().ResourcesFor(securityv1.SchemeGroupVersion.WithResource("securitycontextconstraints"))

	if err != nil {
		if apimeta.IsNoMatchError(err) {
			return false, nil
		}
		return false, fmt.Errorf("could not determine if running on OCP/OKD: %w", err)
	}

	return true, nil
}

// IsOpenShiftFromEnv returns true if the IS_OPENSHIFT environment variable
// is set to "true". The operator sets this variable on all handler-deployed
// pods so they can skip the API server discovery call at startup.
func IsOpenShiftFromEnv() bool {
	return os.Getenv("IS_OPENSHIFT") == "true"
}

// FetchOpenShiftTLSOpts detects OpenShift and fetches the TLS profile for
// secure serving. Returns nil on non-OpenShift clusters.
func FetchOpenShiftTLSOpts(cfg *rest.Config, scheme *runtime.Scheme) (func(*tls.Config), error) {
	kclient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed creating client for TLS profile detection: %w", err)
	}

	isOCP, err := IsOpenShift(kclient)
	if err != nil {
		return nil, fmt.Errorf("could not determine if running on OpenShift: %w", err)
	}
	if !isOCP {
		return nil, nil
	}

	tlsProfileSpec, err := tlspkg.FetchAPIServerTLSProfile(context.Background(), kclient)
	if err != nil {
		return nil, fmt.Errorf("unable to get TLS profile from API server: %w", err)
	}

	tlsOpts, unsupportedCiphers := tlspkg.NewTLSConfigFromProfile(tlsProfileSpec)
	if len(unsupportedCiphers) > 0 {
		log.Info("TLS configuration contains unsupported ciphers that will be ignored",
			"unsupportedCiphers", unsupportedCiphers)
	}

	return tlsOpts, nil
}

// FetchTLSProfileFromFile reads a TLS profile spec from a JSON file (mounted
// from a ConfigMap) and returns the TLS options function and the raw spec.
// This avoids calling the API server at startup, which is critical when
// network connectivity may be temporarily unavailable.
func FetchTLSProfileFromFile(path string) (func(*tls.Config), configv1.TLSProfileSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, configv1.TLSProfileSpec{}, fmt.Errorf("failed reading TLS profile from %s: %w", path, err)
	}

	var spec configv1.TLSProfileSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, configv1.TLSProfileSpec{}, fmt.Errorf("failed parsing TLS profile from %s: %w", path, err)
	}

	tlsOpts, unsupportedCiphers := tlspkg.NewTLSConfigFromProfile(spec)
	if len(unsupportedCiphers) > 0 {
		log.Info("TLS configuration contains unsupported ciphers that will be ignored",
			"unsupportedCiphers", unsupportedCiphers)
	}

	return tlsOpts, spec, nil
}
