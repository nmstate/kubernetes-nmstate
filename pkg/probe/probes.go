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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	yaml "sigs.k8s.io/yaml"

	"github.com/tidwall/gjson"

	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
)

var (
	log = logf.Log.WithName("probe")
)

type Probe struct {
	name      string
	timeout   time.Duration
	condition func(client.Client, time.Duration) wait.ConditionWithContextFunc
}

type Route struct {
	nextHop net.IP
	iface   string
}

const (
	defaultGwRetrieveTimeout  = 120 * time.Second
	defaultGwProbeTimeout     = 120 * time.Second
	defaultDNSProbeTimeout    = 120 * time.Second
	apiServerProbeTimeout     = 120 * time.Second
	nodeReadinessProbeTimeout = 120 * time.Second
	mainRoutingTableID        = 254
	ProbesTotalTimeout        = defaultGwRetrieveTimeout +
		defaultDNSProbeTimeout +
		defaultDNSProbeTimeout +
		apiServerProbeTimeout +
		nodeReadinessProbeTimeout
)

func currentStateAsGJson() (gjson.Result, error) {
	observedStateRaw, err := nmstatectl.Show()
	if err != nil {
		return gjson.Result{}, errors.Wrap(err, "failed retrieving current state")
	}
	res, err := yamlToGJson(observedStateRaw)
	if err != nil {
		return gjson.Result{}, errors.Wrap(err, "failed to convert the current state to GJson")
	}
	return res, nil
}

func yamlToGJson(rawYaml string) (gjson.Result, error) {
	json, err := yaml.YAMLToJSON([]byte(rawYaml))
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(json), nil
}

func apiServerCondition(_ client.Client, _ time.Duration) wait.ConditionWithContextFunc {
	return checkAPIServerConnectivity
}

// This probes use its own client to bypass cache
func checkAPIServerConnectivity(ctx context.Context) (bool, error) {
	// Create new custom client to bypass cache [1]
	// [1] https://github.com/operator-framework/operator-sdk/blob/master/doc/user/client.md#non-default-client
	currentConfig, err := config.GetConfig()
	if err != nil {
		return false, errors.Wrap(err, "getting config")
	}

	// Disable client request timeout
	currentConfig.Timeout = 0
	cli, err := client.New(currentConfig, client.Options{})
	if err != nil {
		log.Error(err, "failed to creating new custom client")
		return false, nil
	}
	err = cli.Get(ctx, types.NamespacedName{Name: metav1.NamespaceDefault}, &corev1.Namespace{})
	if err != nil {
		log.Error(err, "failed reaching the apiserver")
		return false, nil
	}
	return true, nil
}

func nodeReadinessCondition(cli client.Client, _ time.Duration) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (bool, error) {
		return checkNodeReadiness(ctx, cli)
	}
}

func checkNodeReadiness(ctx context.Context, cli client.Client) (bool, error) {
	nodeName := environment.NodeName()
	node := corev1.Node{}
	err := cli.Get(ctx, types.NamespacedName{Name: nodeName}, &node)
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
}

func defaultGws(currentState gjson.Result) ([]Route, error) {
	var found []Route
	currentState.Get("routes.running").ForEach(
		func(_, v gjson.Result) bool {
			// we want to pick the next hops related to the "main" table because we may have multiple tables
			if (v.Get("destination").String() == "0.0.0.0/0" || v.Get("destination").String() == "::/0") &&
				v.Get("table-id").Int() == mainRoutingTableID {
				found = append(found, Route{
					nextHop: net.ParseIP(v.Get("next-hop-address").String()),
					iface:   v.Get("next-hop-interface").String(),
				})
			}
			return true
		},
	)

	if len(found) == 0 {
		msg := "default gw missing"
		defaultGwLog := log.WithValues("path", "routes.running.next-hop-address", "table-id", mainRoutingTableID)
		defaultGwLogDebug := defaultGwLog.V(1)
		if defaultGwLogDebug.Enabled() {
			defaultGwLogDebug.Info(msg, "state", currentState.String())
		} else {
			defaultGwLog.Info(msg)
		}
		return nil, errors.New(msg)
	}
	return found, nil
}

func pingConditionForGateway(gwIP net.IP) func(client.Client, time.Duration) wait.ConditionWithContextFunc {
	return func(_ client.Client, _ time.Duration) wait.ConditionWithContextFunc {
		return func(context.Context) (bool, error) {
			// Re-discover the current route for this gateway IP so that the interface
			// is up to date (e.g. after NNCP moves eth0 under a bridge or bond).
			gjsonCurrentState, err := currentStateAsGJson()
			if err != nil {
				log.Error(err, "failed retrieving current state for ping probe")
				return false, nil
			}
			gws, err := defaultGws(gjsonCurrentState)
			if err != nil {
				log.Error(err, "failed to retrieve default gateways for ping probe")
				return false, nil
			}
			for _, gw := range gws {
				if gw.nextHop.Equal(gwIP) {
					pingOutput, err := ping(gw)
					if err != nil {
						log.Error(err, fmt.Sprintf("error pinging gateway %s via %s -> output: '%s'", gw.nextHop, gw.iface, pingOutput))
						return false, nil
					}
					return true, nil
				}
			}
			log.Info(fmt.Sprintf("gateway %s no longer in routing table", gwIP))
			return false, nil
		}
	}
}

func selectPingProbes(ctx context.Context, cli client.Client) []Probe {
	var probes []Probe

	gjsonCurrentState, err := currentStateAsGJson()
	if err != nil {
		log.Error(err, "failed retrieving current state for ping probe gateway discovery")
		return probes
	}

	gws, err := defaultGws(gjsonCurrentState)
	if err != nil {
		log.Info(fmt.Sprintf("WARNING: no default gateways found for ping probes: %v", err))
		return probes
	}

	for _, gw := range gws {
		p := Probe{
			name:      fmt.Sprintf("ping-%s", gw.nextHop),
			timeout:   defaultGwProbeTimeout,
			condition: pingConditionForGateway(gw.nextHop),
		}
		err := wait.PollUntilContextTimeout(ctx, time.Second, p.timeout, true, p.condition(cli, p.timeout))
		if err == nil {
			probes = append(probes, p)
		} else {
			log.Info(fmt.Sprintf("WARNING not selecting %s probe", p.name))
		}
	}
	return probes
}

func ping(target Route) (string, error) {
	// If next hop is IPv6 link-local, we need to append an interface otherwise it is
	// not clear which interface should be used for communication (e.g. ping test).
	// As this syntax works always, we simply append it always.
	//
	// It is safe to ignore gosec error about concatenated strings as the Route struct
	// is not directly taking any user input.
	cmd := exec.Command("ping", "-I", target.iface, "-c", "1", target.nextHop.String()) // #nosec G204
	var outputBuffer bytes.Buffer
	cmd.Stdout = &outputBuffer
	cmd.Stderr = &outputBuffer
	err := cmd.Run()
	output := fmt.Sprintf("cmd output: '%s'", outputBuffer.String())
	if err != nil {
		return "", errors.Wrapf(err, "failed running ping probe: %s", output)
	}
	return output, nil
}

func dnsCondition(cli client.Client, timeout time.Duration) wait.ConditionWithContextFunc {
	return func(context.Context) (bool, error) {
		return runDNS(cli, timeout)
	}
}

func runDNS(_ client.Client, timeout time.Duration) (bool, error) {
	runningServersGJsonPath := "dns-resolver.running.server"
	errs := []error{}

	// Get the name servers at node since the ones at container may not be up to date
	currentStateAsGJson, err := currentStateAsGJson()
	if err != nil {
		return false, errors.Wrap(err, "failed retrieving current state to get name resolving config")
	}
	runningNameServers := currentStateAsGJson.Get(runningServersGJsonPath).Array()
	if len(runningNameServers) == 0 {
		log.Info(fmt.Sprintf("missing name servers at '%s' on %s", runningServersGJsonPath, currentStateAsGJson.String()))
		return false, nil
	}
	for _, runningNameServer := range runningNameServers {
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, network, net.JoinHostPort(runningNameServer.String(), "53"))
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)

		// NMstate CRD should have always PROBE_DNS_HOST defaulting to root-servers.net but if that
		// for any reason is not the case, here we are safeguarding against missing env variable.
		dnsTarget := environment.GetEnvVar("PROBE_DNS_HOST", "root-servers.net")
		var err error
		if dnsTarget == "root-servers.net" {
			_, err = r.LookupNS(ctx, dnsTarget)
		} else {
			_, err = r.LookupHost(ctx, dnsTarget)
		}

		if err != nil {
			cancel()
			errs = append(errs, err)
		} else {
			cancel()
			return true, nil
		}
	}
	log.Error(fmt.Errorf("%v", errs), "failed checking DNS connectivity")
	return false, nil
}

// Select will return the external connectivity probes that are working (ping and dns) and
// the internal connectivity probes
func Select(ctx context.Context, cli client.Client) []Probe {
	probes := selectPingProbes(ctx, cli)

	dnsProbe := Probe{
		name:      "dns",
		timeout:   defaultDNSProbeTimeout,
		condition: dnsCondition,
	}
	err := wait.PollUntilContextTimeout(ctx, time.Second, dnsProbe.timeout, true /*immediate*/, dnsProbe.condition(cli, dnsProbe.timeout))
	if err == nil {
		probes = append(probes, dnsProbe)
	} else {
		log.Info(fmt.Sprintf("WARNING not selecting %s probe", dnsProbe.name))
	}

	probes = append(probes,
		Probe{
			name:      "api-server",
			timeout:   apiServerProbeTimeout,
			condition: apiServerCondition,
		},
		Probe{
			name:      "node-readiness",
			timeout:   nodeReadinessProbeTimeout,
			condition: nodeReadinessCondition,
		})

	return probes
}

// Run will run the externalConnectivityProbes and also some internal
// kubernetes cluster connectivity and node readiness probes
func Run(ctx context.Context, cli client.Client, probes []Probe) error {
	currentState, err := nmstatectl.Show()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve currentState at runProbes")
	}

	for _, p := range probes {
		log.Info(fmt.Sprintf("Running '%s' probe", p.name))
		err = wait.PollUntilContextTimeout(ctx, time.Second, p.timeout, true /*immediate*/, p.condition(cli, p.timeout))
		if err != nil {
			return errors.Wrapf(
				err,
				"failed runnig probe '%s' with after network reconfiguration -> currentState: %s", p.name, currentState,
			)
		}
	}
	return nil
}
