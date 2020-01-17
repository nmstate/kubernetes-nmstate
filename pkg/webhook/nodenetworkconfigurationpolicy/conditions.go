package nodenetworkconfigurationpolicy

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	jsonpatchv2 "gomodules.xyz/jsonpatch/v2"

	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

const (
	TimestampLabelKey = "nmstate.io/webhook-mutating-timestamp"
)

var log = logf.Log.WithName("webhook/nodenetworkconfigurationpolicy/conditions")

func resetConditionsPatch() jsonpatchv2.Operation {
	unknownConditions := nmstatev1alpha1.ConditionList{}
	for _, conditionType := range nmstatev1alpha1.NodeNetworkConfigurationPolicyConditionTypes {
		unknownConditions.Set(
			conditionType,
			corev1.ConditionUnknown,
			"", "")
	}
	return jsonpatchv2.Operation{
		Path:      "/status/conditions",
		Operation: "replace",
		Value:     unknownConditions,
	}
}

func mutationAnnotationPatch(annotations map[string]string) jsonpatchv2.Operation {
	value := strconv.FormatInt(time.Now().UnixNano(), 10)
	if annotations == nil || len(annotations) == 0 {
		return jsonpatchv2.Operation{
			Path:      "/metadata/annotations",
			Operation: "add",
			Value: map[string]string{
				TimestampLabelKey: value,
			},
		}
	} else {
		// When using jsonpatch the path has to escape slash [1]
		// [1] http://jsonpatch.com/#json-pointer
		key := strings.ReplaceAll(TimestampLabelKey, "/", "~1")
		return jsonpatchv2.Operation{
			Path:      "/metadata/annotations/" + key,
			Operation: "replace",
			Value:     value,
		}
	}
}

func resetConditions(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	var policy nmstatev1alpha1.NodeNetworkConfigurationPolicy
	err := json.Unmarshal(req.Object.Raw, &policy)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.Patched("Conditions reset",
		resetConditionsPatch(),
		mutationAnnotationPatch(policy.ObjectMeta.Annotations))
}

func resetConditionsHook() *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(resetConditions),
	}
}
