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
	"fmt"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
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

	tlsProfileSpec, err := nmstatetls.FetchAPIServerTLSProfile(context.Background(), kclient)
	if err != nil {
		return nil, fmt.Errorf("unable to get TLS profile from API server: %w", err)
	}

	tlsOpts, unsupportedCiphers := nmstatetls.NewTLSConfigFromProfile(tlsProfileSpec)
	if len(unsupportedCiphers) > 0 {
		log.Info("TLS configuration contains unsupported ciphers that will be ignored",
			"unsupportedCiphers", unsupportedCiphers)
	}

	return tlsOpts, nil
}
