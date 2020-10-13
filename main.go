package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/qinqon/kube-admission-webhook/pkg/certificate"

	"k8s.io/apimachinery/pkg/util/wait"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/nmstate/kubernetes-nmstate/api"
	"github.com/nmstate/kubernetes-nmstate/controllers"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	"github.com/nmstate/kubernetes-nmstate/pkg/webhook"

	"github.com/nightlyone/lockfile"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type ProfilerConfig struct {
	EnableProfiler bool   `envconfig:"ENABLE_PROFILER"`
	ProfilerPort   string `envconfig:"PROFILER_PORT" default:"6060"`
}

var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	var logType string
	// Print V(2) logs from packages using klog
	klog.InitFlags(nil)
	flag.Set("v", "2")

	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())
	pflag.StringVar(&logType, "v", "production", "Log type (debug/production).")
	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	if logType == "debug" {
		logf.SetLogger(logf.ZapLogger(true))
	} else {
		logf.SetLogger(logf.ZapLogger(false))
	}

	printVersion()

	// Lock only for handler, we can run old and new version of
	// webhook without problems, policy status will be updated
	// by multiple instances.
	if environment.IsHandler() {
		handlerLock, err := lockHandler()
		if err != nil {
			log.Error(err, "Failed to run lockHandler")
			os.Exit(1)
		}
		defer handlerLock.Unlock()
		log.Info("Successfully took nmstate exclusive lock")
	}

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	mgrOptions := manager.Options{
		Namespace:      namespace,
		MapperProvider: apiutil.NewDiscoveryRESTMapper,
	}

	// We need to add LeaerElection for the webhook
	// cert-manager
	if environment.IsWebhook() {
		mgrOptions.LeaderElection = true
		mgrOptions.LeaderElectionID = "nmstate-webhook-lock"
		mgrOptions.LeaderElectionNamespace = namespace
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, mgrOptions)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := api.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Runs only webhook controllers if it's specified
	if environment.IsWebhook() {

		webhookOpts := certificate.Options{
			Namespace:   os.Getenv("POD_NAMESPACE"),
			WebhookName: "nmstate",
			WebhookType: certificate.MutatingWebhook,
		}

		webhookOpts.CARotateInterval, err = environment.LookupAsDuration("CA_ROTATE_INTERVAL")
		if err != nil {
			log.Error(err, "Failed retrieving ca rotate interval")
			os.Exit(1)
		}

		webhookOpts.CAOverlapInterval, err = environment.LookupAsDuration("CA_OVERLAP_INTERVAL")
		if err != nil {
			log.Error(err, "Failed retrieving ca overlap interval")
			os.Exit(1)
		}

		webhookOpts.CertRotateInterval, err = environment.LookupAsDuration("CERT_ROTATE_INTERVAL")
		if err != nil {
			log.Error(err, "Failed retrieving cert rotate interval")
			os.Exit(1)
		}

		if err := webhook.AddToManager(mgr, webhookOpts); err != nil {
			log.Error(err, "Cannot initialize webhook")
			os.Exit(1)
		}
	} else {
		// Setup all Controllers
		if err := controller.AddToManager(mgr); err != nil {
			log.Error(err, "Cannot initialize controller")
			os.Exit(1)
		}
	}

	setProfiler()

	log.Info("Starting the Cmd.")
	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

// Start profiler on given port if ENABLE_PROFILER is True
func setProfiler() {
	cfg := ProfilerConfig{}
	envconfig.Process("", &cfg)
	if cfg.EnableProfiler {
		log.Info("Starting profiler")
		go func() {
			profilerAddress := fmt.Sprintf("0.0.0.0:%s", cfg.ProfilerPort)
			log.Info(fmt.Sprintf("Starting Profiler Server! \t Go to http://%s/debug/pprof/\n", profilerAddress))
			err := http.ListenAndServe(profilerAddress, nil)
			if err != nil {
				log.Info("Failed to start the server! Error: %v", err)
			}
		}()
	}
}

func lockHandler() (lockfile.Lockfile, error) {
	lockFilePath, ok := os.LookupEnv("NMSTATE_INSTANCE_NODE_LOCK_FILE")
	if !ok {
		return "", errors.New("Failed to find NMSTATE_INSTANCE_NODE_LOCK_FILE ENV var")
	}
	log.Info(fmt.Sprintf("Try to take exclusive lock on file: %s", lockFilePath))
	handlerLock, err := lockfile.New(lockFilePath)
	if err != nil {
		return handlerLock, errors.Wrapf(err, "failed to create lockFile for %s", lockFilePath)
	}
	err = wait.PollImmediateInfinite(5*time.Second, func() (done bool, err error) {
		err = handlerLock.TryLock()
		if err != nil {
			log.Error(err, "retrying to lock handler")
			return false, nil // Don't return the error here, it will not re-poll if we do
		}
		return true, nil
	})
	return handlerLock, err
}
