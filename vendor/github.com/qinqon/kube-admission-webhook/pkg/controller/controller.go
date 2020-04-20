package controller

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type Controller interface {
	controller.Controller
	NeedLeaderElection() bool
}

type Options struct {
	// MaxConcurrentReconciles is the maximum number of concurrent Reconciles which can be run. Defaults to 1.
	MaxConcurrentReconciles int

	// Reconciler reconciles an object
	Reconciler reconcile.Reconciler

	WithoutLeaderElection bool
}

func (o Options) crOptions() controller.Options {
	return controller.Options{
		MaxConcurrentReconciles: o.MaxConcurrentReconciles,
		Reconciler:              o.Reconciler,
	}
}

type leaderElectionAwareController struct {
	ctrl               controller.Controller
	needLeaderElection bool
}

func (c *leaderElectionAwareController) Watch(src source.Source, eventhandler handler.EventHandler, predicates ...predicate.Predicate) error {
	return c.ctrl.Watch(src, eventhandler, predicates...)
}
func (c *leaderElectionAwareController) Start(stop <-chan struct{}) error {
	return c.ctrl.Start(stop)
}

func (c *leaderElectionAwareController) NeedLeaderElection() bool {
	return c.needLeaderElection
}

func (c *leaderElectionAwareController) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	return c.ctrl.Reconcile(request)
}

func (c *leaderElectionAwareController) InjectFunc(f inject.Func) error {
	if _, err := inject.InjectorInto(f, c.ctrl); err != nil {
		return err
	}
	return nil
}

// Fake mgr is needed since it's impossible to create a controller-runtime
// controller without being added to a manager for 0.4.0 version at master
// this is already fixed with a NewUnmanaged constructor, this will be not
// needed
type managerNoopAdd struct{ mgr manager.Manager }

func (m *managerNoopAdd) SetFields(f interface{}) error {
	return m.mgr.SetFields(f)
}
func (m *managerNoopAdd) AddHealthzCheck(name string, check healthz.Checker) error {
	return m.mgr.AddHealthzCheck(name, check)
}
func (m *managerNoopAdd) AddReadyzCheck(name string, check healthz.Checker) error {
	return m.mgr.AddReadyzCheck(name, check)
}
func (m *managerNoopAdd) Start(ch <-chan struct{}) error {
	return m.mgr.Start(ch)
}
func (m *managerNoopAdd) GetConfig() *rest.Config {
	return m.mgr.GetConfig()
}
func (m *managerNoopAdd) GetScheme() *runtime.Scheme {
	return m.mgr.GetScheme()
}
func (m *managerNoopAdd) GetClient() client.Client {
	return m.mgr.GetClient()
}
func (m *managerNoopAdd) GetFieldIndexer() client.FieldIndexer {
	return m.mgr.GetFieldIndexer()
}
func (m *managerNoopAdd) GetCache() cache.Cache {
	return m.mgr.GetCache()
}
func (m *managerNoopAdd) GetEventRecorderFor(name string) record.EventRecorder {
	return m.mgr.GetEventRecorderFor(name)
}
func (m *managerNoopAdd) GetRESTMapper() meta.RESTMapper {
	return m.mgr.GetRESTMapper()
}
func (m *managerNoopAdd) GetAPIReader() client.Reader {
	return m.mgr.GetAPIReader()
}
func (m *managerNoopAdd) GetWebhookServer() *webhook.Server {
	return m.mgr.GetWebhookServer()
}

// Bypass add that's what we want this manager for since we cannot
// construct a new Controller without Add being called
func (m *managerNoopAdd) Add(manager.Runnable) error {
	return nil
}

func New(name string, mgr manager.Manager, options Options) (Controller, error) {
	ctrl, err := controller.New(name, &managerNoopAdd{mgr: mgr}, options.crOptions())
	if err != nil {
		return nil, err
	}
	controllerWrapper := &leaderElectionAwareController{
		ctrl:               ctrl,
		needLeaderElection: !options.WithoutLeaderElection,
	}
	return controllerWrapper, mgr.Add(controllerWrapper)
}
