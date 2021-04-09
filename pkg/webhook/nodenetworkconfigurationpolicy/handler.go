package nodenetworkconfigurationpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

type mutator func(nmstatev1beta1.NodeNetworkConfigurationPolicy) nmstatev1beta1.NodeNetworkConfigurationPolicy
type validator func(nmstatev1beta1.NodeNetworkConfigurationPolicy, nmstatev1beta1.NodeNetworkConfigurationPolicy) []metav1.StatusCause

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

func validatePolicyHandler(cli client.Client, neededValidationFor func(nmstatev1beta1.NodeNetworkConfigurationPolicy, nmstatev1beta1.NodeNetworkConfigurationPolicy) bool, validators ...validator) admission.HandlerFunc {
	log := logf.Log.WithName("webhook/nodenetworkconfigurationpolicy/validator")
	return func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
		original := req.Object.Raw
		policy := nmstatev1beta1.NodeNetworkConfigurationPolicy{}
		err := json.Unmarshal(original, &policy)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, errors.Wrapf(err, "failed decoding policy: %s", string(original)))
		}
		currentPolicy := nmstatev1beta1.NodeNetworkConfigurationPolicy{}
		err = cli.Get(context.TODO(), types.NamespacedName{Name: policy.GetName(), Namespace: policy.GetNamespace()}, &currentPolicy)
		if err != nil && !apierrors.IsNotFound(err) {
			errMsg := fmt.Sprintf("failed getting policy %s", string(original))
			log.Error(err, errMsg)
			return admission.Errored(http.StatusInternalServerError, errors.Wrapf(err, errMsg))
		}

		if !neededValidationFor(policy, currentPolicy) {
			return admission.Allowed("validation not needed")
		}

		errCauses := []metav1.StatusCause{}
		for _, validate := range validators {
			errCauses = append(errCauses, validate(policy, currentPolicy)...)
		}
		if len(errCauses) > 0 {
			return admission.Denied(handlePolicyCauses(errCauses, policy.Name))
		}
		return admission.Allowed("")
	}
}

func handlePolicyCauses(causes []metav1.StatusCause, name string) string {
	errMsg := fmt.Sprintf("failed to admit NodeNetworkConfigurationPolicy %s: ", name)
	for _, cause := range causes {
		errMsg += fmt.Sprintf("message: %s. ", cause.Message)
	}
	return errMsg
}

func always(nmstatev1beta1.NodeNetworkConfigurationPolicy) bool {
	return true
}
