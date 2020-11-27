package probe

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	yaml "sigs.k8s.io/yaml"

	"github.com/tidwall/gjson"

	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
)

var (
	log = logf.Log.WithName("probe")
)

type Probe struct {
	name    string
	timeout time.Duration
	run     func(client.Client, time.Duration) error
}

const (
	defaultGwRetrieveTimeout  = 120 * time.Second
	defaultGwProbeTimeout     = 120 * time.Second
	defaultDnsProbeTimeout    = 120 * time.Second
	apiServerProbeTimeout     = 120 * time.Second
	nodeReadinessProbeTimeout = 120 * time.Second
)

func currentStateAsGJson() (gjson.Result, error) {
	observedStateRaw, err := nmstatectl.Show()
	if err != nil {
		return gjson.Result{}, errors.Wrap(err, "failed retrieving current state")
	}

	currentState, err := yaml.YAMLToJSON([]byte(observedStateRaw))
	if err != nil {
		return gjson.Result{}, errors.Wrap(err, "failed to convert current state to JSON")
	}
	return gjson.ParseBytes(currentState), nil

}

func ping(target string, timeout time.Duration) (string, error) {
	output := ""
	return output, wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		cmd := exec.Command("ping", "-c", "1", target)
		var outputBuffer bytes.Buffer
		cmd.Stdout = &outputBuffer
		cmd.Stderr = &outputBuffer
		err := cmd.Run()
		output = fmt.Sprintf("cmd output: '%s'", outputBuffer.String())
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

// This probes use its own client to bypass cache that
// why we wrap it to ignore the one it's passed
func checkApiServerConnectivity(timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		// Create new custom client to bypass cache [1]
		// [1] https://github.com/operator-framework/operator-sdk/blob/master/doc/user/client.md#non-default-client
		config, err := config.GetConfig()
		if err != nil {
			return false, errors.Wrap(err, "getting config")
		}
		// Since we are going to retrieve Nodes default schema is good
		// enough, also align timeout with poll
		config.Timeout = timeout
		client, err := client.New(config, client.Options{})
		if err != nil {
			log.Error(err, "failed to creating new custom client")
			return false, nil
		}
		err = client.Get(context.TODO(), types.NamespacedName{Name: metav1.NamespaceDefault}, &corev1.Namespace{})
		if err != nil {
			log.Error(err, "failed reaching the apiserver")
			return false, nil
		}
		return true, nil
	})
}

func checkNodeReadiness(client client.Client, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		nodeName := environment.NodeName()
		node := corev1.Node{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: nodeName}, &node)
		if err != nil {
			return false, errors.Wrapf(err, "failed retrieving pod's node %s", nodeName)
		}
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady &&
				condition.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
}

func defaultGw() (string, error) {
	defaultGw := ""
	return defaultGw, wait.PollImmediate(time.Second, defaultGwRetrieveTimeout, func() (bool, error) {
		gjsonCurrentState, err := currentStateAsGJson()
		if err != nil {
			return false, errors.Wrap(err, "failed retrieving current state to retrieve default gw")
		}
		defaultGwGjsonPath := "routes.running.#(destination==\"0.0.0.0/0\").next-hop-address"
		defaultGw = gjsonCurrentState.
			Get(defaultGwGjsonPath).String()
		if defaultGw == "" {
			msg := "default gw missing"
			defaultGwLog := log.WithValues("path", defaultGwGjsonPath)
			defaultGwLogDebug := defaultGwLog.V(1)
			if defaultGwLogDebug.Enabled() {
				defaultGwLogDebug.Info(msg, "state", gjsonCurrentState.String())
			} else {
				defaultGwLog.Info(msg)
			}
			return false, nil
		}

		return true, nil
	})
}

func runPing(client client.Client, timeout time.Duration) error {
	defaultGw, err := defaultGw()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve default gw at runProbes")
	}

	pingOutput, err := ping(defaultGw, timeout)
	if err != nil {
		return errors.Wrapf(err, "error pinging default gateway -> output: %s", pingOutput)
	}
	return nil
}
func lookupRootNS(nameServer string, timeout time.Duration) error {
	rootNS := "root-server.net"
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: timeout,
			}
			return d.DialContext(ctx, network, net.JoinHostPort(nameServer, "53"))
		},
	}
	// We use a closure to create a scope for defer here
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := r.LookupNS(ctx, rootNS)
	if err != nil {
		return errors.Wrapf(err, "failed looking up NS %s using name sever %s", rootNS, nameServer)
	}
	return nil
}

func runDNS(client client.Client, timeout time.Duration) error {
	currentStateAsGJson, err := currentStateAsGJson()
	if err != nil {
		return errors.Wrap(err, "failed retrieving current state to get name resolving config")
	}

	// Get the name servers at node since the ones at container are not accurate
	runningServersGJsonPath := "dns-resolver.running.server"
	runningNameServers := currentStateAsGJson.Get(runningServersGJsonPath).Array()
	if len(runningNameServers) == 0 {
		return fmt.Errorf("missing name servers at '%s' on %s", runningServersGJsonPath, currentStateAsGJson.String())
	}

	errs := []error{}
	for _, runningNameServer := range runningNameServers {
		err = lookupRootNS(runningNameServer.String(), defaultDnsProbeTimeout)
		if err != nil {
			errs = append(errs, err)
		} else {
			return nil
		}
	}
	return fmt.Errorf("failed checking DNS connectivity: %v", errs)
}

// Select will return the external connectivity probes that are working (ping and dns) and
// the internal connectivity probes
func Select(cli client.Client) []Probe {
	probes := []Probe{}

	err := runPing(cli, time.Second)
	if err == nil {
		probes = append(probes, Probe{
			name:    "ping",
			timeout: defaultGwProbeTimeout,
			run:     runPing,
		})
	} else {
		log.Info("WARNING not selecting 'ping' probe")
	}
	err = runDNS(cli, time.Second)
	if err == nil {
		probes = append(probes, Probe{
			name:    "dns",
			timeout: defaultDnsProbeTimeout,
			run:     runDNS,
		})
	} else {
		log.Info("WARNING not selecting 'dns' probe")
	}

	probes = append(probes, Probe{
		name:    "api-server",
		timeout: apiServerProbeTimeout,
		run: func(_ client.Client, timeout time.Duration) error {
			return checkApiServerConnectivity(timeout)
		},
	})

	probes = append(probes, Probe{
		name:    "node-readiness",
		timeout: nodeReadinessProbeTimeout,
		run:     checkNodeReadiness,
	})

	return probes
}

// Run will run the externalConnectivityProbes and also some internal
// kubernetes cluster connectivity and node readiness probes
func Run(client client.Client, probes []Probe) error {
	currentState, err := nmstatectl.Show()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve currentState at runProbes")
	}

	for _, p := range probes {
		log.Info(fmt.Sprintf("Running '%s' probe", p.name))
		err = p.run(client, p.timeout)
		if err != nil {
			return errors.Wrapf(err, "failed runnig probe '%s' with after network reconfiguration -> currentState: %s", p.name, currentState)
		}

	}
	return nil
}
