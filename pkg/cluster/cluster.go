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
	"fmt"

	securityv1 "github.com/openshift/api/security/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
