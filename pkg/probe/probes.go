package probe

import (
	"bytes"
	"context"
	"fmt"
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

const (
	defaultGwRetrieveTimeout  = 120 * time.Second
	defaultGwProbeTimeout     = 120 * time.Second
	apiServerProbeTimeout     = 120 * time.Second
	nodeReadinessProbeTimeout = 120 * time.Second
)

func ping(target string, timeout time.Duration) (string, error) {
	output := ""
	return output, wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
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

func checkApiServerConnectivity(timeout time.Duration) error {
	return wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
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
	return wait.PollImmediate(1*time.Second, timeout, func() (bool, error) {
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
	return defaultGw, wait.PollImmediate(1*time.Second, defaultGwRetrieveTimeout, func() (bool, error) {
		observedStateRaw, err := nmstatectl.Show()
		if err != nil {
			log.Error(err, fmt.Sprintf("failed retrieving current state"))
			return false, nil
		}

		currentState, err := yaml.YAMLToJSON([]byte(observedStateRaw))
		if err != nil {
			return false, errors.Wrap(err, "failed to convert current state to JSON")
		}

		defaultGw = gjson.ParseBytes(currentState).
			Get("routes.running.#(destination==\"0.0.0.0/0\").next-hop-address").String()
		if defaultGw == "" {
			log.Info("default gw missing", "state", string(currentState))
			return false, nil
		}

		return true, nil
	})
}

func RunAll(client client.Client) error {
	defaultGw, err := defaultGw()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve default gw at runProbes")
	}

	currentState, err := nmstatectl.Show()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve currentState at runProbes")
	}

	// TODO: Make ping timeout configurable with a config map
	pingOutput, err := ping(defaultGw, defaultGwProbeTimeout)
	if err != nil {
		return errors.Wrapf(err, "error pinging external address after network reconfiguration -> output: %s, currentState: %s", pingOutput, currentState)
	}

	err = checkApiServerConnectivity(apiServerProbeTimeout)
	if err != nil {
		return errors.Wrapf(err, "error checking api server connectivity after network reconfiguration -> currentState: %s", currentState)
	}

	err = checkNodeReadiness(client, nodeReadinessProbeTimeout)
	if err != nil {
		return errors.Wrapf(err, "error checking node readiness after network reconfiguration -> currentState: %s", currentState)
	}

	return nil
}
