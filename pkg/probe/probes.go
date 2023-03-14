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
	condition func(client.Client) wait.ConditionFunc
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

var nmstatectlShow = func() (string, error) {
	return nmstatectl.Show()
}

func currentStateAsGJson() (gjson.Result, error) {
	observedStateRaw, err := nmstatectlShow()
	if err != nil {
		return gjson.Result{}, errors.Wrap(err, "failed retrieving current state")
	}

	currentState, err := yaml.YAMLToJSON([]byte(observedStateRaw))
	if err != nil {
		return gjson.Result{}, errors.Wrap(err, "failed to convert current state to JSON")
	}
	return gjson.ParseBytes(currentState), nil
}

func apiServerCondition(_ client.Client) wait.ConditionFunc {
	return checkAPIServerConnectivity
}

// This probes use its own client to bypass cache
func checkAPIServerConnectivity() (bool, error) {
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
	err = cli.Get(context.TODO(), types.NamespacedName{Name: metav1.NamespaceDefault}, &corev1.Namespace{})
	if err != nil {
		log.Error(err, "failed reaching the apiserver")
		return false, nil
	}
	return true, nil
}

func nodeReadinessCondition(cli client.Client) wait.ConditionFunc {
	return func() (bool, error) {
		return checkNodeReadiness(cli)
	}
}

func checkNodeReadiness(cli client.Client) (bool, error) {
	nodeName := environment.NodeName()
	node := corev1.Node{}
	err := cli.Get(context.TODO(), types.NamespacedName{Name: nodeName}, &node)
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

// defaultGw gets the current nmstate in JSON format with library tidwall/gjson. It then filters the JSON according to
// the following criteria:
// * Get all routes with table-id==254, table-id==0 or table-id not set (the best way to find unset fields with gjson
// is to convert the field to boolean. Boolean false will match table-id==0 or an empty table-id field).
// * Then get the first route for all matching table-ids with destination==0.0.0.0/0.
// See here for more details https://github.com/tidwall/gjson/issues/212 and
// https://github.com/tidwall/gjson/blob/master/SYNTAX.md#queries for more details.
func defaultGw() (string, error) {
	gjsonCurrentState, err := currentStateAsGJson()
	if err != nil {
		return "", errors.Wrap(err, "failed retrieving current state to retrieve default gw")
	}
	defaultGwGjsonPath := fmt.Sprintf("[routes.running.#(table-id==~false)#,routes.running.#(table-id==%d)#].@flatten"+
		"|#(destination==\"0.0.0.0/0\").next-hop-address", mainRoutingTableID)
	defaultGw := gjsonCurrentState.Get(defaultGwGjsonPath).String()
	if defaultGw == "" {
		msg := "default gw missing"
		defaultGwLog := log.WithValues("path", defaultGwGjsonPath)
		defaultGwLogDebug := defaultGwLog.V(1)
		if defaultGwLogDebug.Enabled() {
			defaultGwLogDebug.Info(msg, "state", gjsonCurrentState.String())
		} else {
			defaultGwLog.Info(msg)
		}
		return "", errors.New(msg)
	}
	return defaultGw, nil
}

func pingCondition(cli client.Client) wait.ConditionFunc {
	return func() (bool, error) {
		return runPing(cli)
	}
}

func runPing(_ client.Client) (bool, error) {
	defaultGw, err := defaultGw()
	if err != nil {
		log.Error(err, "failed to retrieve default gw")
		return false, nil
	}

	pingOutput, err := ping(defaultGw)
	if err != nil {
		log.Error(err, fmt.Sprintf("error pinging default gateway -> output: '%s'", pingOutput))
		return false, nil
	}
	return true, nil
}

func ping(target string) (string, error) {
	cmd := exec.Command("ping", "-c", "1", target)
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

func dnsCondition(cli client.Client) wait.ConditionFunc {
	return func() (bool, error) {
		return runDNS(cli)
	}
}

func runDNS(_ client.Client) (bool, error) {
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
		_, err := r.LookupNS(context.TODO(), "root-server.net")
		if err != nil {
			errs = append(errs, err)
		} else {
			return true, nil
		}
	}
	log.Error(fmt.Errorf("%v", errs), "failed checking DNS connectivity")
	return false, nil
}

// Select will return the external connectivity probes that are working (ping and dns) and
// the internal connectivity probes
func Select(cli client.Client) []Probe {
	probes := []Probe{}
	externalConnectivityProbes := []Probe{
		{
			name:      "ping",
			timeout:   defaultGwProbeTimeout,
			condition: pingCondition,
		},
		{
			name:      "dns",
			timeout:   defaultDNSProbeTimeout,
			condition: dnsCondition,
		},
	}

	for _, p := range externalConnectivityProbes {
		err := wait.PollImmediate(time.Second, p.timeout, p.condition(cli))
		if err == nil {
			probes = append(probes, p)
		} else {
			log.Info(fmt.Sprintf("WARNING not selecting %s probe", p.name))
		}
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
func Run(cli client.Client, probes []Probe) error {
	currentState, err := nmstatectlShow()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve currentState at runProbes")
	}

	for _, p := range probes {
		log.Info(fmt.Sprintf("Running '%s' probe", p.name))
		err = wait.PollImmediate(time.Second, p.timeout, p.condition(cli))
		if err != nil {
			return errors.Wrapf(
				err,
				"failed runnig probe '%s' with after network reconfiguration -> currentState: %s", p.name, currentState,
			)
		}
	}
	return nil
}
