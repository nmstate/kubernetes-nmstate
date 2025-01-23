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
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// +kubebuilder:scaffold:imports

	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/pflag"

	openshiftconsolev1 "github.com/openshift/api/console/v1"
	openshiftoperatorv1 "github.com/openshift/api/operator/v1"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	nmstatev1alpha1 "github.com/nmstate/kubernetes-nmstate/api/v1alpha1"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	controllers "github.com/nmstate/kubernetes-nmstate/controllers/operator"
)

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
	utilruntime.Must(openshiftoperatorv1.Install(scheme))
	utilruntime.Must(openshiftconsolev1.Install(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
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

	ctrlOptions := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0", // disable metrics
		},
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrlOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	err = setupOperatorController(mgr)
	if err != nil {
		setupLog.Error(err, "unable to setup controller", "controller", "NMState")
		os.Exit(1)
	}

	setProfiler()

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupOperatorController(mgr manager.Manager) error {
	apiClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme(), Mapper: mgr.GetRESTMapper()})
	if err != nil {
		return fmt.Errorf("failed creating non cached client: %w", err)
	}

	if err = (&controllers.NMStateReconciler{
		Client:    mgr.GetClient(),
		APIClient: apiClient,
		Log:       ctrl.Log.WithName("controllers").WithName("NMState"),
		Scheme:    mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("failed creating NMState CR controller: %w", err)
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
