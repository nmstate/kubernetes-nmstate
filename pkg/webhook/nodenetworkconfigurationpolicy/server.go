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

package nodenetworkconfigurationpolicy

import (
	"crypto/tls"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func Add(mgr manager.Manager) error {
	// We need two hooks, the update of nncp and nncp/status (it's a subresource) happens
	// at different times, also if you modify status at nncp webhook it does not modify it,
	// so you need nncp/status webhook that will catch that and do the final modifications.
	// So this works this way:
	// 1.- User changes nncp desiredState so it triggers deleteConditionsHook()
	// 2.- Since we have deleted the condition the status-mutate webhook is called and
	//     there we set conditions to Unknown. This final result will be updated.
	server := webhook.NewServer(webhook.Options{
		// Disable HTTP2 to avoid CVE-2023-39325
		TLSOpts: []func(config *tls.Config){
			func(c *tls.Config) {
				c.NextProtos = []string{"http/1.1"}
			},
		},
	},
	)
	server.Register("/readyz", healthz.CheckHandler{Checker: healthz.Ping})
	server.Register("/nodenetworkconfigurationpolicies-mutate", deleteConditionsHook())
	server.Register("/nodenetworkconfigurationpolicies-status-mutate", setConditionsUnknownHook())
	server.Register("/nodenetworkconfigurationpolicies-timestamp-mutate", setTimestampAnnotationHook())
	server.Register("/nodenetworkconfigurationpolicies-update-validate", validatePolicyUpdateHook(mgr.GetClient()))
	server.Register("/nodenetworkconfigurationpolicies-create-validate", validatePolicyCreateHook(mgr.GetClient()))
	return mgr.Add(server)
}
