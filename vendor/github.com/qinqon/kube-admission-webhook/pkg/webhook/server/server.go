package server

import (
	"crypto/x509"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/qinqon/kube-admission-webhook/pkg/webhook/server/certificate"
	"github.com/qinqon/kube-admission-webhook/pkg/webhook/server/certificate/triple"
)

type Server struct {
	webhookServer *webhook.Server
	certManager   *certificate.Manager
	log           logr.Logger
}

type ServerModifier func(w *webhook.Server)

// Add creates a new Conditions Mutating Webhook and adds it to the Manager. The Manager will set fields on the Webhook
// and Start it when the Manager is Started.
func New(client client.Client, webhookName string, webhookType certificate.WebhookType, serverOpts ...ServerModifier) *Server {
	s := &Server{
		webhookServer: &webhook.Server{
			Port:    8443,
			CertDir: "/etc/webhook/certs/",
		},
		certManager: certificate.NewManager(client, webhookName, webhookType, certificate.OneYearDuration),
		log:         logf.Log.WithName("webhook/server"),
	}
	s.UpdateOpts(serverOpts...)
	s.webhookServer.Register("/readyz", healthz.CheckHandler{Checker: healthz.Ping})
	return s
}

func WithHook(path string, hook *webhook.Admission) ServerModifier {
	return func(s *webhook.Server) {
		s.Register(path, hook)
	}
}

func WithPort(port int) ServerModifier {
	return func(s *webhook.Server) {
		s.Port = port
	}
}

func WithCertDir(certDir string) ServerModifier {
	return func(s *webhook.Server) {
		s.CertDir = certDir
	}
}

//updates Server parameters using ServerModifier functions. Once the manager is started these parameters cannot be updated
func (s *Server) UpdateOpts(serverOpts ...ServerModifier) {
	for _, serverOpt := range serverOpts {
		serverOpt(s.webhookServer)
	}
}

func (s *Server) Add(mgr manager.Manager) error {
	err := mgr.Add(s.certManager)
	if err != nil {
		return errors.Wrap(err, "failed adding certificate manager to controller-runtime manager")
	}
	err = mgr.Add(s)
	if err != nil {
		return errors.Wrap(err, "failed adding webhook server to controller-runtime manager")
	}
	return nil
}

func (s *Server) checkTLS() error {

	_, err := os.Stat(path.Join(s.webhookServer.CertDir, corev1.TLSPrivateKeyKey))
	if err != nil {
		return errors.Wrap(err, "failed checking TLS key file stats")
	}

	_, err = os.Stat(path.Join(s.webhookServer.CertDir, corev1.TLSCertKey))
	if err != nil {
		return errors.Wrap(err, "failed checking TLS cert file stats")
	}

	key, err := ioutil.ReadFile(path.Join(s.webhookServer.CertDir, corev1.TLSPrivateKeyKey))
	if err != nil {
		return errors.Wrap(err, "failed reading for TLS key")
	}

	_, err = triple.ParsePrivateKeyPEM(key)
	if err != nil {
		return errors.Wrap(err, "failed parsing TLS key")
	}

	certPEM, err := ioutil.ReadFile(path.Join(s.webhookServer.CertDir, corev1.TLSCertKey))
	if err != nil {
		return errors.Wrap(err, "failed reading for TLS cert")
	}

	certs, err := triple.ParseCertsPEM(certPEM)
	if err != nil {
		return errors.Wrap(err, "failed parsing TLS cert")
	}

	caPEM, err := s.certManager.CABundle()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve CA cert")
	}

	cas := x509.NewCertPool()
	ok := cas.AppendCertsFromPEM([]byte(caPEM))
	if !ok {
		return errors.New("failed to parse CA certificate")
	}

	opts := x509.VerifyOptions{
		Roots:   cas,
		DNSName: certs[0].DNSNames[0],
	}

	if _, err := certs[0].Verify(opts); err != nil {
		return errors.Wrap(err, "failed to verify certificate")
	}

	return nil
}

func (s *Server) waitForTLSReadiness() error {
	return wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		err := s.checkTLS()
		if err != nil {
			utilruntime.HandleError(err)
			return false, nil
		}
		return true, nil
	})
}

func (s *Server) Start(stop <-chan struct{}) error {
	s.log.Info("Starting nodenetworkconfigurationpolicy webhook server")

	err := s.waitForTLSReadiness()
	if err != nil {
		return errors.Wrap(err, "failed watting for ready TLS key/cert")
	}

	err = s.webhookServer.Start(stop)
	if err != nil {
		return errors.Wrap(err, "failed starting webhook server")
	}
	return nil
}

func (s *Server) InjectFunc(f inject.Func) error {
	return s.webhookServer.InjectFunc(f)
}

func (s *Server) NeedLeaderElection() bool {
	return s.webhookServer.NeedLeaderElection()
}
