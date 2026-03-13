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
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

type mutator func(*nmstatev1.NodeNetworkConfigurationPolicy)

func mutatePolicyHandler(neededMutationFor func(*nmstatev1.NodeNetworkConfigurationPolicy) bool, mutate mutator) admission.HandlerFunc {
	log := logf.Log.WithName("webhook/nodenetworkconfigurationpolicy/mutator")
	return func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
		original := req.Object.Raw
		policy := nmstatev1.NodeNetworkConfigurationPolicy{}
		err := json.Unmarshal(original, &policy)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, errors.Wrapf(err, "failed decoding policy: %s", string(original)))
		}

		if !neededMutationFor(&policy) {
			return admission.Allowed("mutation not needed")
		}

		mutate(&policy)
		current, err := json.Marshal(policy)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, errors.Wrapf(err, "failed encoding policy: %+v", policy))
		}

		response := admission.PatchResponseFromRaw(original, current)
		log.Info(fmt.Sprintf("webhook response: %+v", response))
		return response
	}
}

func always(*nmstatev1.NodeNetworkConfigurationPolicy) bool {
	return true
}
