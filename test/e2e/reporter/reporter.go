package e2e

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
)

type KubernetesNMStateReporter struct {
	artifactsDir         string
	namespace            string
	previousDeviceStatus string
	nodes                []string
}

func New(artifactsDir string, namespace string, nodes []string) *KubernetesNMStateReporter {
	return &KubernetesNMStateReporter{
		artifactsDir: artifactsDir,
		namespace:    namespace,
		nodes:        nodes,
	}
}

func (r *KubernetesNMStateReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {
}

func (r *KubernetesNMStateReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {
	r.Cleanup()
}

func (r *KubernetesNMStateReporter) SpecWillRun(specSummary *types.SpecSummary) {
	if specSummary.Skipped() || specSummary.Pending() {
		return
	}

	r.storeStateBeforeEach()
}
func (r *KubernetesNMStateReporter) SpecDidComplete(specSummary *types.SpecSummary) {
	if specSummary.Skipped() || specSummary.Pending() {
		return
	}

	since := time.Now().Add(-specSummary.RunTime).Add(-5 * time.Second)
	name := strings.Join(specSummary.ComponentTexts[1:], " ")
	passed := specSummary.Passed()

	r.dumpStateAfterEach(name, since, passed)
}

func (r *KubernetesNMStateReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
}

func (r *KubernetesNMStateReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {
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
		names, err := ioutil.ReadDir(r.artifactsDir)
		if err != nil {
			panic(err)
		}
		for _, entery := range names {
			os.RemoveAll(path.Join([]string{r.artifactsDir, entery.Name()}...))
		}
	}
}

func (r *KubernetesNMStateReporter) logNetworkManager(testName string, sinceTime time.Time) {
	r.OpenTestLogFile("NetworkManager", testName, networkManagerLogsWriter(r.nodes, sinceTime))
}

func (r *KubernetesNMStateReporter) logPods(testName string, sinceTime time.Time) error {
	if framework.Global.LocalOperator {
		return nil
	}

	// Let's print the pods logs to the GinkgoWriter so
	// we see the failure directly at prow junit output without opening files
	r.OpenTestLogFile("pods", testName, podLogsWriter(r.namespace, sinceTime), GinkgoWriter)

	return nil
}

func (r *KubernetesNMStateReporter) OpenTestLogFile(logType string, testName string, cb func(f io.Writer), extraWriters ...io.Writer) {
	name := fmt.Sprintf("%s/%s_%s.log", r.artifactsDir, testName, logType)
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
