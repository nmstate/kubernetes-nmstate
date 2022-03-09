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

//go:build cgo

package nmstatectl

import (
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	nmstateapi "github.com/nmstate/kubernetes-nmstate/api/shared"
	"nmstate.io/go/nmstate"
)

func Show() (string, error) {
	return nmstate.New().RetrieveNetState()
}

func Set(desiredState nmstateapi.State, timeout time.Duration) (string, error) {
	var setDoneCh = make(chan struct{})
	go setUnavailableUp(setDoneCh)
	defer close(setDoneCh)

	stateJSON, err := yaml.YAMLToJSON([]byte(desiredState.Raw))
	if err != nil {
		return "", err
	}

	nmstateLog := strings.Builder{}
	nmstatectl := nmstate.New(nmstate.WithLogsWritter(&nmstateLog), nmstate.WithTimeout(timeout), nmstate.WithNoCommit())
	output, err := nmstatectl.ApplyNetState(string(stateJSON))
	if err != nil {
		return nmstateLog.String(), err
	}
	nmstateLog.WriteString(output)
	return nmstateLog.String(), err
}

func Commit() (string, error) {
	return nmstate.New().CommitCheckpoint("")
}

func Rollback() (string, error) {
	return nmstate.New().RollbackCheckpoint("")
}
