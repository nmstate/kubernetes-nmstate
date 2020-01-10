package mutating

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	webhookName = "nmstate"
)

type server struct {
	mgr           manager.Manager
	webhookServer *webhook.Server
	log           logr.Logger
}

// Add creates a new Conditions Mutating Webhook and adds it to the Manager. The Manager will set fields on the Webhook
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newServer(mgr))
}

func newServer(mgr manager.Manager) *server {
	s := &server{
		webhookServer: &webhook.Server{
			Port:    8443,
			CertDir: "/etc/webhook/certs/",
		},
		mgr: mgr,
		log: logf.Log.WithName("webhook/mutating/server"),
	}
	s.webhookServer.Register("/nodenetworkconfigurationpolicies-mutate", resetConditionsHook())
	return s
}

// add adds a new Webhook to mgr with r as the webhook.Server
func add(mgr manager.Manager, s *server) error {
	mgr.Add(s)
	return nil
}

// Retrieve cluster CA bundle and encode to base 64
func (s *server) clientCAFile() ([]byte, error) {
	authenticationConfig := corev1.ConfigMap{}
	err := s.mgr.GetClient().Get(context.TODO(), types.NamespacedName{Namespace: "kube-system", Name: "extension-apiserver-authentication"}, &authenticationConfig)

	if err != nil {
		return []byte{}, errors.Wrap(err, "failed to retrieve cluster authentication config")
	}
	clientCaFile := authenticationConfig.Data["client-ca-file"]
	return []byte(clientCaFile), nil
}

func (s *server) updateCABundle() error {
	s.log.Info("Updating CA bundle for webhook")
	mutatingWebHook := admissionregistrationv1beta1.MutatingWebhookConfiguration{}

	clientCAFile, err := s.clientCAFile()
	if err != nil {
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Do some polling to wait for manifest to be deployed
		err := wait.PollImmediate(1*time.Second, 120*time.Second, func() (bool, error) {
			webhookKey := types.NamespacedName{Name: webhookName}
			err := s.mgr.GetClient().Get(context.TODO(), webhookKey, &mutatingWebHook)
			if err != nil {
				return false, err
			}
			return true, nil
		})

		if err != nil {
			return errors.Wrap(err, "failed retrieving mutationg webhook "+webhookName)
		}
		// If CA bundle is already there, we are good and finish
		if len(mutatingWebHook.Webhooks[0].ClientConfig.CABundle) > 0 {
			s.log.Info("CA bundle already set")
			return nil
		}

		// Update the CA bundle at webhook
		mutatingWebHook.Webhooks[0].ClientConfig.CABundle = []byte(clientCAFile)
		err = s.mgr.GetClient().Update(context.TODO(), &mutatingWebHook)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to update mutating webhook CABundle")
	}
	return nil
}

func (s *server) Start(stop <-chan struct{}) error {
	err := s.updateCABundle()
	if err != nil {
		return errors.Wrap(err, "failed updating CA bundle at webhook")
	}
	return s.webhookServer.Start(stop)
}

func (s *server) InjectFunc(f inject.Func) error {
	return s.webhookServer.InjectFunc(f)
}

func (s *server) NeedLeaderElection() bool {
	return s.webhookServer.NeedLeaderElection()
}
