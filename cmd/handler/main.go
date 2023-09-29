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

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	// +kubebuilder:scaffold:imports

	"github.com/gofrs/flock"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/qinqon/kube-admission-webhook/pkg/certificate"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/nmstate/kubernetes-nmstate/api/names"
	nmstateapi "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/api/v1alpha1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	controllers "github.com/nmstate/kubernetes-nmstate/controllers/handler"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	"github.com/nmstate/kubernetes-nmstate/pkg/file"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
	"github.com/nmstate/kubernetes-nmstate/pkg/webhook"
)

const generalExitStatus int = 1

type ProfilerConfig struct {
	EnableProfiler bool   `envconfig:"ENABLE_PROFILER"`
	ProfilerPort   string `envconfig:"PROFILER_PORT" default:"6060"`
}

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nmstatev1.AddToScheme(scheme))
	utilruntime.Must(nmstatev1beta1.AddToScheme(scheme))
	utilruntime.Must(nmstatev1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	if mainHandler() == generalExitStatus {
		os.Exit(generalExitStatus)
	}
}

// The code from main() has to be extracted into another function in order to properly handle defer.
// Otherwise, defer may never execute because of eventual os.Exit().
// return 1 indicates that program should exit with status code 1
func mainHandler() int {
	opt := zap.Options{}
	opt.BindFlags(flag.CommandLine)
	var logType string
	pflag.StringVar(&logType, "v", "production", "Log type (debug/production).")
	pflag.CommandLine.MarkDeprecated("v", "please use the --zap-devel flag for debug logging instead")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	if logType == "debug" {
		// workaround until --v flag got removed
		flag.CommandLine.Set("zap-devel", "true")
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opt)))
	// Lock only for handler, we can run old and new version of
	// webhook without problems, policy status will be updated
	// by multiple instances.
	if environment.IsHandler() {
		handlerLock, err := lockHandler()
		if err != nil {
			setupLog.Error(err, "Failed to run lockHandler")
			return generalExitStatus
		}
		defer handlerLock.Unlock()
		setupLog.Info("Successfully took nmstate exclusive lock")
	}

	ctrlOptions := ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0", // disable metrics
	}

	if environment.IsHandler() {
		cacheResourcesOnNodes(&ctrlOptions)
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrlOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return generalExitStatus
	}

	if environment.IsCertManager() {
		var certManagerOpts certificate.Options
		if certManagerOpts, err = retrieveCertAndCAIntervals(); err != nil {
			return generalExitStatus
		}
		if err = setupCertManager(mgr, certManagerOpts); err != nil {
			return generalExitStatus
		}
		// Runs only webhook controllers if it's specified
	} else if environment.IsWebhook() {
		if err = webhook.AddToManager(mgr); err != nil {
			setupLog.Error(err, "Cannot initialize webhook")
			return generalExitStatus
		}
	} else if environment.IsHandler() {
		if err = setupHandlerControllers(mgr); err != nil {
			return generalExitStatus
		}
		if err = checkNmstateIsWorking(); err != nil {
			return generalExitStatus
		}
		if err = createHealthyFile(); err != nil {
			return generalExitStatus
		}
	}

	setProfiler()

	setupLog.Info("starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return generalExitStatus
	}

	return 0
}

// Handler runs as a daemonset and we want that each handler pod will cache/reconcile only resources that belong the node it runs on.
func cacheResourcesOnNodes(ctrlOptions *ctrl.Options) {
	nodeName := environment.NodeName()
	metadataNameMatchingNodeNameSelector := fields.Set{"metadata.name": nodeName}.AsSelector()
	nodeLabelMatchingNodeNameSelector := labels.Set{nmstateapi.EnactmentNodeLabel: nodeName}.AsSelector()
	ctrlOptions.NewCache = cache.BuilderWithOptions(cache.Options{
		SelectorsByObject: cache.SelectorsByObject{
			&corev1.Node{}: {
				Field: metadataNameMatchingNodeNameSelector,
			},
			&nmstatev1beta1.NodeNetworkState{}: {
				Field: metadataNameMatchingNodeNameSelector,
			},
			&nmstatev1beta1.NodeNetworkConfigurationEnactment{}: {
				Label: nodeLabelMatchingNodeNameSelector,
			},
		},
	})
}

func setupHandlerControllers(mgr manager.Manager) error {
	if err := (&controllers.NodeReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Node"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create Node controller", "controller", "NMState")
		return err
	}

	apiClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme(), Mapper: mgr.GetRESTMapper()})
	if err != nil {
		setupLog.Error(err, "failed creating non cached client")
		return err
	}

	if err = (&controllers.NodeNetworkConfigurationPolicyReconciler{
		Client:    mgr.GetClient(),
		APIClient: apiClient,
		Log:       ctrl.Log.WithName("controllers").WithName("NodeNetworkConfigurationPolicy"),
		Scheme:    mgr.GetScheme(),
		Recorder:  mgr.GetEventRecorderFor(fmt.Sprintf("%s.nmstate-handler", environment.NodeName())),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create NodeNetworkConfigurationPolicy controller", "controller", "NMState")
		return err
	}

	if err = (&controllers.NodeNetworkConfigurationEnactmentReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NodeNetworkConfigurationEnactment"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create NodeNetworkConfigurationEnactment controller", "controller", "NMState")
		return err
	}

	return nil
}

// Handler runs with host networking so opening ports is problematic
// they will collide with node ports so to ensure that we reach this
// point (we have the handler lock and nmstatectl show is working) a
// file is touched and the file is checked at readinessProbe field.
func createHealthyFile() error {
	healthyFile := "/tmp/healthy"
	setupLog.Info("Marking handler as healthy touching healthy file", "healthyFile", healthyFile)
	err := file.Touch(healthyFile)
	if err != nil {
		setupLog.Error(err, "failed marking handler as healthy")
		return err
	}
	return nil
}

func checkNmstateIsWorking() error {
	_, err := nmstatectl.Show()
	if err != nil {
		setupLog.Error(err, "failed checking nmstatectl health")
		return err
	}
	return nil
}

func retrieveCertAndCAIntervals() (certificate.Options, error) {
	certManagerOpts := certificate.Options{
		Namespace:   os.Getenv("POD_NAMESPACE"),
		WebhookName: "nmstate",
		WebhookType: certificate.MutatingWebhook,
		ExtraLabels: names.IncludeRelationshipLabels(nil),
	}

	var err error
	certManagerOpts.CARotateInterval, err = environment.LookupAsDuration("CA_ROTATE_INTERVAL")
	if err != nil {
		setupLog.Error(err, "Failed retrieving ca rotate interval")
		return certificate.Options{}, err
	}

	certManagerOpts.CAOverlapInterval, err = environment.LookupAsDuration("CA_OVERLAP_INTERVAL")
	if err != nil {
		setupLog.Error(err, "Failed retrieving ca overlap interval")
		return certificate.Options{}, err
	}

	certManagerOpts.CertRotateInterval, err = environment.LookupAsDuration("CERT_ROTATE_INTERVAL")
	if err != nil {
		setupLog.Error(err, "Failed retrieving cert rotate interval")
		return certificate.Options{}, err
	}

	certManagerOpts.CertOverlapInterval, err = environment.LookupAsDuration("CERT_OVERLAP_INTERVAL")
	if err != nil {
		setupLog.Error(err, "Failed retrieving cert overlap interval")
		return certificate.Options{}, err
	}

	return certManagerOpts, nil
}

func setupCertManager(mgr manager.Manager, certManagerOpts certificate.Options) error {
	certManager, err := certificate.NewManager(mgr.GetClient(), &certManagerOpts)
	if err != nil {
		setupLog.Error(err, "unable to create cert-manager", "controller", "cert-manager")
		return err
	}
	err = certManager.Add(mgr)
	if err != nil {
		setupLog.Error(err, "unable to add cert-manager to controller-runtime manager", "controller", "cert-manager")
		return err
	}
	return nil
}

// Start profiler on given port if ENABLE_PROFILER is True
func setProfiler() {
	cfg := ProfilerConfig{}
	envconfig.Process("", &cfg)
	if cfg.EnableProfiler {
		setupLog.Info("Starting profiler")
		go func() {
			profilerAddress := fmt.Sprintf("0.0.0.0:%s", cfg.ProfilerPort)
			setupLog.Info(fmt.Sprintf("Starting Profiler Server! \t Go to http://%s/debug/pprof/\n", profilerAddress))
			server := &http.Server{ReadHeaderTimeout: 10 * time.Second, Addr: profilerAddress}
			err := server.ListenAndServe()
			if err != nil {
				setupLog.Info("Failed to start the server! Error: %v", err)
			}
		}()
	}
}

func lockHandler() (*flock.Flock, error) {
	lockFilePath, ok := os.LookupEnv("NMSTATE_INSTANCE_NODE_LOCK_FILE")
	if !ok {
		return nil, errors.New("Failed to find NMSTATE_INSTANCE_NODE_LOCK_FILE ENV var")
	}
	setupLog.Info(fmt.Sprintf("Try to take exclusive lock on file: %s", lockFilePath))
	handlerLock := flock.New(lockFilePath)
	interval := 5 * time.Second
	err := wait.PollUntilContextCancel(context.TODO(), interval, true, /*immediate*/
		func(context.Context) (done bool, err error) {
			locked, err := handlerLock.TryLock()
			if err != nil {
				setupLog.Error(err, "retrying to lock handler")
				return false, nil // Don't return the error here, it will not re-poll if we do
			}
			return locked, nil
		})
	return handlerLock, err
}
