package nodenetworkconfigurationpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func mutateAllPoliciesHandler(cli client.Client, neededMutationFor func(oldNode, newNode corev1.Node) bool, mutators ...mutator) admission.HandlerFunc {
	log := logf.Log.WithName("webhook/nodenetworkconfigurationpolicy/mutatePolicyOnNodeModifiedHandler")
	return func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
		oldNode, err := decodeNode(req.OldObject)
		if err != nil {
			log.Error(err, "failed decoding original node")
			return admission.Allowed("")
		}
		newNode, err := decodeNode(req.Object)
		if err != nil {
			log.Error(err, "failed decoding new node")
			return admission.Allowed("")
		}
		if !neededMutationFor(oldNode, newNode) {
			return admission.Allowed("mutation not needed")
		}

		policyList := nmstatev1beta1.NodeNetworkConfigurationPolicyList{}
		err = cli.List(context.TODO(), &policyList)
		if err != nil {
			log.Error(err, "failed listing all NodeNetworkConfigurationPolicies to reset their Status after node modified ")
			return admission.Allowed("")
		}

		for _, policy := range policyList.Items {
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				err := cli.Get(context.TODO(), types.NamespacedName{Name: policy.Name}, &policy)
				if err != nil {
					return err
				}
				for _, mutate := range mutators {
					policy = mutate(policy)
				}
				err = cli.Status().Update(context.TODO(), &policy)
				if err != nil {
					return err
				}
				return cli.Update(context.TODO(), &policy)
			})
			if err != nil {
				log.Error(err, "failed mutating NNCP after a node change detected")
			}
		}

		return admission.Allowed("")
	}
}

func decodeNode(nodeObject runtime.RawExtension) (corev1.Node, error) {
	node := corev1.Node{}
	err := json.Unmarshal(nodeObject.Raw, &node)
	if err != nil {
		return node, errors.Wrapf(err, "failed decoding node: %s", string(nodeObject.Raw))
	}
	return node, nil
}
