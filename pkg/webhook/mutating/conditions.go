package mutating

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	jsonpatchv2 "gomodules.xyz/jsonpatch/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1alpha1"
)

const (
	TimestampLabelKey = "nmstate.io/webhook-mutating-timestamp"
)

var log = logf.Log.WithName("webhook")

// Add creates a new Conditions Mutating Webhook and adds it to the Manager. The Manager will set fields on the Webhook
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newServer())
}

func newServer() *webhook.Server {
	server := &webhook.Server{
		Port:    8443,
		CertDir: "/etc/webhook/certs/",
	}

	log.Info(fmt.Sprintf("Registering mutating webhook at %+v", server))
	server.Register("/mutating", resetConditionsHook())
	return server
}

// add adds a new Webhook to mgr with r as the webhook.Server
func add(mgr manager.Manager, s *webhook.Server) error {
	mgr.Add(s)
	return nil
}

func resetConditionsPatch() jsonpatchv2.Operation {
	return jsonpatchv2.Operation{
		Path:      "/status/conditions",
		Operation: "replace",
		Value:     nmstatev1alpha1.ConditionList{},
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
