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

package reporter

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2/types"
)

type KubernetesNMStateReporter struct {
	artifactsDir         string
	namespace            string
	previousDeviceStatus string
	nodes                []string
}

func New(artifactsDir, namespace string, nodes []string) *KubernetesNMStateReporter {
	return &KubernetesNMStateReporter{
		artifactsDir: artifactsDir,
		namespace:    namespace,
		nodes:        nodes,
	}
}

func (r *KubernetesNMStateReporter) ReportBeforeEach(specReport types.SpecReport) {
	if specReport.State.Is(types.SpecStateSkipped) || specReport.State.Is(types.SpecStatePending) {
		return
	}

	r.storeStateBeforeEach()
}

func (r *KubernetesNMStateReporter) ReportAfterEach(specReport types.SpecReport) {
	if specReport.State.Is(types.SpecStateSkipped) || specReport.State.Is(types.SpecStatePending) {
		return
	}

	since := time.Now().Add(-specReport.RunTime).Add(-5 * time.Second)
	name := strings.Join(specReport.ContainerHierarchyTexts, " ")
	passed := specReport.State.Is(types.SpecStatePassed)

	r.dumpStateAfterEach(name, since, passed)
}

func (r *KubernetesNMStateReporter) storeStateBeforeEach() {
	r.previousDeviceStatus = r.deviceStatus()
}

func runAndWait(funcs ...func()) {
	var wg sync.WaitGroup
	wg.Add(len(funcs))
	for _, f := range funcs {
		// You have to pass f to the goroutine, it's going to change
		// at the next loop iteration.
		go func(rf func()) {
			rf()
			wg.Done()
		}(f)
	}
	wg.Wait()
}

func (r *KubernetesNMStateReporter) dumpStateAfterEach(testName string, testStartTime time.Time, passed bool) {
	if passed {
		return
	}
	runAndWait(
		func() { r.logPods(testName, testStartTime) },
		func() { r.logDeviceStatus(testName) },
		func() { r.logNetworkManager(testName, testStartTime) },
	)
}

func (r *KubernetesNMStateReporter) deviceStatus() string {
	stringBuilder := strings.Builder{}
	writeDeviceStatus(&stringBuilder, r.nodes)
	return stringBuilder.String()
}

func (r *KubernetesNMStateReporter) logDeviceStatus(testName string) {
	r.OpenTestLogFile("deviceStatus", testName, func(w io.Writer) {
		writeMessage(w, banner("DEVICE STATUS BEFORE TEST"))
		writeMessage(w, r.previousDeviceStatus)
		writeMessage(w, banner("DEVICE STATUS AFTER TEST"))
		writeDeviceStatus(w, r.nodes)
	})
}

// Cleanup cleans up the current content of the artifactsDir
func (r *KubernetesNMStateReporter) Cleanup() {
	// clean up artifacts from previous run
	if r.artifactsDir != "" {
		_, err := os.Stat(r.artifactsDir)
		if err != nil {
			if os.IsNotExist(err) {
				return
			} else {
				panic(err)
			}
		}
		names, err := os.ReadDir(r.artifactsDir)
		if err != nil {
			panic(err)
		}
		for _, entry := range names {
			os.RemoveAll(path.Join([]string{r.artifactsDir, entry.Name()}...))
		}
	}
}

func (r *KubernetesNMStateReporter) logNetworkManager(testName string, sinceTime time.Time) {
	r.OpenTestLogFile("NetworkManager", testName, networkManagerLogsWriter(r.nodes, sinceTime))
}

func (r *KubernetesNMStateReporter) logPods(testName string, sinceTime time.Time) {
	// Let's print the pods logs to the GinkgoWriter so
	// we see the failure directly at prow junit output without opening files
	r.OpenTestLogFile("pods", testName, podLogsWriter(r.namespace, sinceTime))
}

func (r *KubernetesNMStateReporter) OpenTestLogFile(logType, testName string, cb func(f io.Writer), extraWriters ...io.Writer) {
	testLogDir := filepath.Join(r.artifactsDir, strings.ReplaceAll(testName, " ", "_"))
	err := os.MkdirAll(testLogDir, 0755)
	if err != nil {
		fmt.Println(err)
		return
	}

	name := filepath.Join(testLogDir, fmt.Sprintf("%s.log", logType))
	fi, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := fi.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	writers := []io.Writer{fi}
	if len(extraWriters) > 0 {
		writers = append(writers, extraWriters...)
	}
	cb(io.MultiWriter(writers...))
}
