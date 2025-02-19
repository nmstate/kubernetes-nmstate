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
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
)

type mutator func(*nmstatev1.NodeNetworkConfigurationPolicy)
type validator func(*nmstatev1.NodeNetworkConfigurationPolicy, *nmstatev1.NodeNetworkConfigurationPolicy) []metav1.StatusCause

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

func validatePolicyHandler(
	cli client.Client,
	neededValidationFor func(admissionv1.Operation, *nmstatev1.NodeNetworkConfigurationPolicy, *nmstatev1.NodeNetworkConfigurationPolicy) bool,
	validators ...validator,
) admission.HandlerFunc {
	log := logf.Log.WithName("webhook/nodenetworkconfigurationpolicy/validator")
	return func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
		original := req.Object.Raw
		policy := nmstatev1.NodeNetworkConfigurationPolicy{}
		err := json.Unmarshal(original, &policy)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, errors.Wrapf(err, "failed decoding policy: %s", string(original)))
		}
		currentPolicy := nmstatev1.NodeNetworkConfigurationPolicy{}
		err = cli.Get(context.TODO(), types.NamespacedName{Name: policy.GetName(), Namespace: policy.GetNamespace()}, &currentPolicy)
		if err != nil && !apierrors.IsNotFound(err) {
			errMsg := fmt.Sprintf("failed getting policy %s", string(original))
			log.Error(err, errMsg)
			return admission.Errored(http.StatusInternalServerError, errors.Wrap(err, errMsg))
		}

		if !neededValidationFor(req.Operation, &policy, &currentPolicy) {
			return admission.Allowed("validation not needed")
		}

		errCauses := []metav1.StatusCause{}
		for _, validate := range validators {
			errCauses = append(errCauses, validate(&policy, &currentPolicy)...)
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

func always(*nmstatev1.NodeNetworkConfigurationPolicy) bool {
	return true
}
