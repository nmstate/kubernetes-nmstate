package nmstatectl

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"sigs.k8s.io/yaml"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	"github.com/nmstate/kubernetes-nmstate/pkg/ionmstate"
)

func Show() (string, error) {
	c, err := ionmstate.NewConnection()
	if err != nil {
		return "", errors.Wrapf(err, "failed connecting to varlink to call nmstate show")
	}
	defer c.Close()

	state, logs, err := ionmstate.Show().Call(context.Background(), c, nil)
	logsAsString := ionmstate.ConvertLogsToString(logs)
	if err != nil {
		return "", fmt.Errorf("failed nmstate show: %w, %s", err, logsAsString)
	}
	return string(*state), nil
}

func Set(desiredState nmstate.State, timeout time.Duration) (string, error) {

	desiredStateJson, err := yaml.YAMLToJSON(desiredState.Raw)
	if err != nil {
		return "", fmt.Errorf("failed converting yaml desired state to json: %w, %s", err, desiredState)
	}

	var setDoneCh = make(chan struct{})
	go setUnavailableUp(setDoneCh)
	defer close(setDoneCh)

	c, err := ionmstate.NewConnection()
	if err != nil {
		return "", errors.Wrapf(err, "failed connecting to varlink to call nmstate set")
	}
	defer c.Close()
	arguments := map[string]json.RawMessage{
		"desired_state":    json.RawMessage(desiredStateJson),
		"commit":           json.RawMessage("false"),
		"rollback_timeout": json.RawMessage(strconv.Itoa(int(timeout.Seconds()))),
	}
	logs, err := ionmstate.Apply().Call(context.Background(), c, arguments)
	logsAsString := ionmstate.ConvertLogsToString(logs)
	if err != nil {
		return logsAsString, fmt.Errorf("failed nmstate set arguments(%s): %w, %s", arguments, err, logsAsString)
	}
	return logsAsString, nil
}

func Commit() (string, error) {
	c, err := ionmstate.NewConnection()
	if err != nil {
		return "", errors.Wrapf(err, "failed connecting to varlink nmstate commit")
	}
	defer c.Close()

	logs, err := ionmstate.Commit().Call(context.Background(), c, nil)
	logsAsString := ionmstate.ConvertLogsToString(logs)
	if err != nil {
		return "", fmt.Errorf("failed nmstate commit: %w, %s", err, logsAsString)
	}
	return logsAsString, err
}

func Rollback() error {
	c, err := ionmstate.NewConnection()
	if err != nil {
		return errors.Wrapf(err, "failed connecting to varlink to call nmstate rollback")
	}
	defer c.Close()

	logs, err := ionmstate.Rollback().Call(context.Background(), c, nil)
	logsAsString := ionmstate.ConvertLogsToString(logs)
	if err != nil {
		return fmt.Errorf("failed nmstate rollback: %w, %s", err, logsAsString)
	}
	return err
}
