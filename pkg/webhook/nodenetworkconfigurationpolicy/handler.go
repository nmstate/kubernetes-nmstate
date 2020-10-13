package nodenetworkconfigurationpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

type mutator func(nmstatev1beta1.NodeNetworkConfigurationPolicy) nmstatev1beta1.NodeNetworkConfigurationPolicy

func mutatePolicyHandler(neededMutationFor func(nmstatev1beta1.NodeNetworkConfigurationPolicy) bool, mutate mutator) admission.HandlerFunc {
	log := logf.Log.WithName("webhook/nodenetworkconfigurationpolicy/mutator")
	return func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
		original := req.Object.Raw
		policy := nmstatev1beta1.NodeNetworkConfigurationPolicy{}
		err := json.Unmarshal(original, &policy)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, errors.Wrapf(err, "failed decoding policy: %s", string(original)))
		}

		if !neededMutationFor(policy) {
			return admission.Allowed("mutation not needed")
		}

		policy = mutate(policy)
		current, err := json.Marshal(policy)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, errors.Wrapf(err, "failed encoding policy: %+v", policy))
		}

		response := admission.PatchResponseFromRaw(original, current)
		log.Info(fmt.Sprintf("webhook response: %+v", response))
		return response
	}
}

func always(nmstatev1beta1.NodeNetworkConfigurationPolicy) bool {
	return true
}
