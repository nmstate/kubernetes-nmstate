package nodenetworkconfigurationpolicy

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pkg/errors"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	webhookserver "github.com/qinqon/kube-admission-webhook/pkg/webhook/server"
	"github.com/qinqon/kube-admission-webhook/pkg/webhook/server/certificate"
)

const (
	webhookName = "nmstate"
)

func Add(mgr manager.Manager) error {

	webhookPortString, isSet := os.LookupEnv("WEBHOOK_PORT")
	if !isSet || len(webhookPortString) == 0 {
		return fmt.Errorf("WEBHOOK_PORT env var is mandatory")
	}

	var err error
	webhookPort, err := strconv.Atoi(webhookPortString)
	if err != nil {
		return errors.Wrap(err, "WEBHOOK_PORT env var has bad format")
	}

	// We need two hooks, the update of nncp and nncp/status (it's a subresource) happends
	// at different times, also if you modify status at nncp webhook it does not modify it
	// you need at nncp/status webhook that will catch that and do the final modifications.
	// So this works this way:
	// 1.- User changes nncp desiredState so it triggers deleteConditionsHook()
	// 2.- Since we have delete the condition the status-mutate webhook get called and
	//     there we set conditions to Unknown this final result will be updated.
	server := webhookserver.NewWithAutoCACert(mgr, webhookName, certificate.MutatingWebhook,
		webhookserver.WithPort(webhookPort),
		webhookserver.WithHook("/nodenetworkconfigurationpolicies-mutate", deleteConditionsHook()),
		webhookserver.WithHook("/nodenetworkconfigurationpolicies-status-mutate", setConditionsUnknownHook()),
		webhookserver.WithHook("/nodenetworkconfigurationpolicies-timestamp-mutate", setTimestampAnnotationHook()),
	)
	return add(mgr, server)
}

// add adds a new Webhook to mgr with r as the webhook.Server
func add(mgr manager.Manager, s manager.Runnable) error {
	mgr.Add(s)
	return nil
}
