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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nmstatetls "github.com/nmstate/kubernetes-nmstate/pkg/tls"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("cluster")

// IsOpenShift returns true if the current cluster is an OpenShift/OKD cluster.
func IsOpenShift(kclient client.Client) (bool, error) {
	// if the cluster has the securityContextConstraint resource of the group security.openshift.io, then it is most likely an OCP/OKD cluster
	sccGVR := schema.GroupVersion{Group: "security.openshift.io", Version: "v1"}.WithResource("securitycontextconstraints")
	_, err := kclient.RESTMapper().ResourcesFor(sccGVR)

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

// FetchTLSProfileFromFile reads a TLS profile spec from a JSON file (mounted
// from a ConfigMap) and returns the TLS options function and the raw spec.
// This avoids calling the API server at startup, which is critical when
// network connectivity may be temporarily unavailable.
func FetchTLSProfileFromFile(path string) (func(*tls.Config), nmstatetls.TLSProfileSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nmstatetls.TLSProfileSpec{}, fmt.Errorf("failed reading TLS profile from %s: %w", path, err)
	}

	var spec nmstatetls.TLSProfileSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, nmstatetls.TLSProfileSpec{}, fmt.Errorf("failed parsing TLS profile from %s: %w", path, err)
	}

	tlsOpts, unsupported, err := nmstatetls.NewTLSConfigFromProfile(spec)
	if err != nil {
		return nil, nmstatetls.TLSProfileSpec{}, fmt.Errorf("TLS profile from %s cannot be honored: %w", path, err)
	}
	if !unsupported.IsEmpty() {
		log.Info("TLS profile contains unsupported entries that will be ignored",
			"details", unsupported.Message())
	}

	return tlsOpts, spec, nil
}
