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

	rbac "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	// +kubebuilder:scaffold:imports

	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/pflag"

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
		Scheme:             scheme,
		MetricsBindAddress: "0", // disable metrics
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrlOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.NMStateReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("NMState"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create NMState controller", "controller", "NMState")
		os.Exit(1)
	}

	setProfiler()

	if err := createClusterRole(mgr.GetClient(), mgr.GetAPIReader()); err != nil {
		setupLog.Error(err, "unable to create NMState cluster-reader ClusterRole")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
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
			err := http.ListenAndServe(profilerAddress, nil)
			if err != nil {
				setupLog.Info("Failed to start the server! Error: %v", err)
			}
		}()
	}
}

func createClusterRole(c client.Client, reader client.Reader) error {
	var clusterReader rbac.ClusterRole
	key := types.NamespacedName{Name: "cluster-reader"}
	err := reader.Get(context.TODO(), key, &clusterReader)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	var clusterRole rbac.ClusterRole
	const clusterRoleName = "k8s-nmstate-project"
	err = reader.Get(context.TODO(), types.NamespacedName{Name: clusterRoleName}, &clusterRole)

	clusterRole = getClusterRoleSpec(clusterRoleName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return c.Create(context.TODO(), &clusterRole)
		}
		return err
	}

	return c.Update(context.TODO(), &clusterRole)
}

func getClusterRoleSpec(name string) rbac.ClusterRole {
	clusterRole := rbac.ClusterRole{}
	clusterRole.APIVersion = "rbac.authorization.k8s.io/v1"
	clusterRole.Kind = "ClusterRole"
	clusterRole.Name = name
	clusterRole.Labels = map[string]string{"rbac.authorization.k8s.io/aggregate-to-cluster-reader": "true"}
	clusterRole.Rules = make([]rbac.PolicyRule, 1)
	clusterRole.Rules[0].APIGroups = []string{"nmstate.io"}
	clusterRole.Rules[0].Resources = []string{
		"nodenetworkstates",
		"nodenetworkconfigurationpolicies",
		"nodenetworkconfigurationenactments"}
	clusterRole.Rules[0].Verbs = []string{
		"get",
		"list",
		"watch"}

	return clusterRole
}
