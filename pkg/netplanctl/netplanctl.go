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

package netplanctl

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
)

var log = logf.Log.WithName("netplanctl")

const (
	netplanBusName     = "io.netplan.Netplan"
	netplanObjectPath  = "/io/netplan/Netplan"
	netplanInterface   = "io.netplan.Netplan"
	netplanConfigIface = "io.netplan.Netplan.Config"

	// Timeout constants
	defaultTimeout       = 30 * time.Second
	applyTimeout         = 120 * time.Second
	infoTimeout          = 10 * time.Second
	timeoutBufferSeconds = 10
)

// NetplanClient handles D-Bus communication with netplan.
type NetplanClient struct {
	conn *dbus.Conn
}

// NewNetplanClient creates a new netplan D-Bus client.
func NewNetplanClient() (*NetplanClient, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to system D-Bus")
	}
	return &NetplanClient{conn: conn}, nil
}

// Close closes the D-Bus connection.
func (c *NetplanClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// getConfigObject calls Config() to get a dynamic config object path.
func (c *NetplanClient) getConfigObject() (dbus.ObjectPath, error) {
	obj := c.conn.Object(netplanBusName, netplanObjectPath)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	call := obj.CallWithContext(ctx, netplanInterface+".Config", 0)
	if call.Err != nil {
		return "", errors.Wrap(call.Err, "failed to call netplan D-Bus Config method")
	}

	var configPath dbus.ObjectPath
	if err := call.Store(&configPath); err != nil {
		return "", errors.Wrap(err, "failed to parse netplan Config response")
	}

	return configPath, nil
}

// Try applies configuration with automatic rollback after timeout.
// This is similar to nmstatectl's checkpoint mechanism.
func (c *NetplanClient) Try(config string, timeoutSeconds uint32) error {
	configPath, err := c.getConfigObject()
	if err != nil {
		return err
	}

	configObj := c.conn.Object(netplanBusName, configPath)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Parse the config to extract the network section.
	// The input may be JSON or YAML with a top-level "network" key.
	var configData map[string]interface{}

	// Try parsing as JSON first
	if err := json.Unmarshal([]byte(config), &configData); err != nil {
		// If JSON fails, try YAML
		if err := yaml.Unmarshal([]byte(config), &configData); err != nil {
			return errors.Wrap(err, "failed to parse netplan configuration as JSON or YAML")
		}
	}

	// Extract the network section
	networkConfig, ok := configData["network"]
	if !ok {
		return errors.New("netplan configuration must have a 'network' top-level key")
	}

	// Convert network section to YAML string for netplan
	networkYAML, err := yaml.Marshal(networkConfig)
	if err != nil {
		return errors.Wrap(err, "failed to marshal network config to YAML")
	}

	// Set the configuration using the D-Bus Set() method.
	// Format: "network=<yaml>" with origin "kubernetes-nmstate".
	configString := fmt.Sprintf("network=%s", string(networkYAML))
	call := configObj.CallWithContext(ctx, netplanConfigIface+".Set", 0, configString, "kubernetes-nmstate")
	if call.Err != nil {
		return errors.Wrap(call.Err, "failed to call netplan D-Bus Set method")
	}

	// Call Try on the config object with timeout for automatic rollback.
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds+timeoutBufferSeconds)*time.Second)
	defer cancel2()

	call = configObj.CallWithContext(ctx2, netplanConfigIface+".Try", 0, timeoutSeconds)
	if call.Err != nil {
		return errors.Wrap(call.Err, "failed to call netplan D-Bus Try method")
	}

	return nil
}

// Apply applies the current netplan configuration.
func (c *NetplanClient) Apply() error {
	obj := c.conn.Object(netplanBusName, netplanObjectPath)

	ctx, cancel := context.WithTimeout(context.Background(), applyTimeout)
	defer cancel()

	call := obj.CallWithContext(ctx, netplanInterface+".Apply", 0)
	if call.Err != nil {
		return errors.Wrap(call.Err, "failed to call netplan D-Bus Apply method")
	}

	return nil
}

// Generate generates backend-specific configuration from netplan YAML.
func (c *NetplanClient) Generate() error {
	obj := c.conn.Object(netplanBusName, netplanObjectPath)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	call := obj.CallWithContext(ctx, netplanInterface+".Generate", 0)
	if call.Err != nil {
		return errors.Wrap(call.Err, "failed to call netplan D-Bus Generate method")
	}

	return nil
}

// Cancel cancels a pending Try operation (like nmstatectl rollback).
func (c *NetplanClient) Cancel() error {
	configPath, err := c.getConfigObject()
	if err != nil {
		// If we can't get config object, there's nothing to cancel
		return nil
	}

	configObj := c.conn.Object(netplanBusName, configPath)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	call := configObj.CallWithContext(ctx, netplanConfigIface+".Cancel", 0)
	if call.Err != nil {
		// Cancel might fail if there's nothing to cancel, which is fine
		return nil
	}

	return nil
}

// Info retrieves netplan daemon information.
func (c *NetplanClient) Info() (map[string]interface{}, error) {
	obj := c.conn.Object(netplanBusName, netplanObjectPath)

	ctx, cancel := context.WithTimeout(context.Background(), infoTimeout)
	defer cancel()

	call := obj.CallWithContext(ctx, netplanInterface+".Info", 0)
	if call.Err != nil {
		return nil, errors.Wrap(call.Err, "failed to call netplan D-Bus Info method")
	}

	var info map[string]interface{}
	if err := call.Store(&info); err != nil {
		return nil, errors.Wrap(err, "failed to parse netplan Info response")
	}

	return info, nil
}

// Status retrieves the current network state using netplan status.
// Uses nsenter to run in the host's mount namespace since the container
// has a different mount namespace and needs access to host's systemd.
func (c *NetplanClient) Status() (string, error) {
	output, err := exec.Command("nsenter", "-t", "1", "-m", "netplan", "status", "-f", "json").CombinedOutput()
	log.V(1).Info("netplan status output", "output", string(output))
	return string(output), err
}

// Show returns the current network state via netplan status.
func Show() (string, error) {
	client, err := NewNetplanClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	return client.Status()
}

// Set applies the desired state with timeout.
func Set(desiredState nmstate.State, timeout time.Duration) (string, error) {
	client, err := NewNetplanClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	netplanConfig := string(desiredState.Raw)
	err = client.Try(netplanConfig, uint32(timeout.Seconds()))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Applied netplan configuration with %d second timeout", int(timeout.Seconds())), nil
}

// Commit confirms the pending configuration.
func Commit() (string, error) {
	client, err := NewNetplanClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	configPath, err := client.getConfigObject()
	if err != nil {
		return "", err
	}

	configObj := client.conn.Object(netplanBusName, configPath)

	ctx, cancel := context.WithTimeout(context.Background(), applyTimeout)
	defer cancel()

	call := configObj.CallWithContext(ctx, netplanConfigIface+".Apply", 0)
	if call.Err != nil {
		return "", errors.Wrap(call.Err, "failed to apply netplan configuration")
	}

	return "Netplan configuration committed successfully", nil
}

// Rollback cancels pending configuration changes.
func Rollback() error {
	client, err := NewNetplanClient()
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Cancel()
}
