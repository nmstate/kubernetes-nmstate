package e2e

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

type KubernetesNMStateReporter struct {
	artifactsDir string
}

func NewKubernetesNMStateReporter(artifactsDir string) *KubernetesNMStateReporter {
	return &KubernetesNMStateReporter{
		artifactsDir: artifactsDir,
	}
}

func (r *KubernetesNMStateReporter) BeforeSuiteDidRun() {
	r.Cleanup()
}

func (r *KubernetesNMStateReporter) dumpStateBeforeEach(testName string) {
	r.logDeviceStatus(testName)
}

func (r *KubernetesNMStateReporter) dumpStateAfterEach(testName string, namespace string, testStartTime time.Time) {
	r.logPods(testName, namespace, testStartTime)
	r.logDeviceStatus(testName)
	r.logNetworkManager(testName, testStartTime)
}

func (r *KubernetesNMStateReporter) logDeviceStatus(testName string) {

	r.OpenTestLogFile("deviceStatus", testName, func(writer io.ReadWriteCloser) {
		writer.Write([]byte(fmt.Sprintf("\n***** Start printing device status *****\n\n")))
		for _, node := range nodes {
			output, err := runAtNode(node, "/usr/bin/nmcli", "c", "s")
			Expect(err).ToNot(HaveOccurred())
			writer.Write([]byte(fmt.Sprintf("\n***** Connection status on node %s *****\n\n %s", node, output)))
			writer.Write([]byte(fmt.Sprintf("\n***** Done Connection status on node %s*****\n", node)))

			output, err = runAtNode(node, "/usr/bin/nmcli", "d", "s")
			Expect(err).ToNot(HaveOccurred())
			writer.Write([]byte(fmt.Sprintf("\n***** Device status on node %s ***** \n\n %s", node, output)))
			writer.Write([]byte(fmt.Sprintf("\n***** Done device status on node %s *****\n", node)))
			output, err = runAtNode(node, "/usr/sbin/ip", "-4", "-o", "a")
			Expect(err).ToNot(HaveOccurred())
			writer.Write([]byte(fmt.Sprintf("\n***** Configured ipv4 ips on devices on node %s *****\n\n %s", node, output)))
			writer.Write([]byte(fmt.Sprintf("\n***** Done ip status on node %s *****\n", node)))
		}
		writer.Write([]byte(fmt.Sprintf("\n***** Finished printing device status *****\n\n")))
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
	r.OpenTestLogFile("NetworkManager", testName, func(writer io.ReadWriteCloser) {
		for _, node := range nodes {
			output, err := runAtNode(node, "sudo", "journalctl", "-u", "NetworkManager",
				"--since", fmt.Sprintf("'%ds ago'", 10+int(time.Now().Sub(sinceTime).Seconds())))
			Expect(err).ToNot(HaveOccurred())
			writer.Write([]byte(fmt.Sprintf("\n***** Journalctl for NetworkManager on node %s *****\n\n %s", node, output)))
			writer.Write([]byte(fmt.Sprintf("\n***** Done NetworkManager logs on node %s *****\n", node)))
		}
	})
}

func (r *KubernetesNMStateReporter) logPods(testName string, namespace string, sinceTime time.Time) error {
	if framework.Global.LocalOperator {
		return nil
	}
	r.OpenTestLogFile("pods", testName, func(writer io.ReadWriteCloser) {
		podLogOpts := corev1.PodLogOptions{}
		podLogOpts.SinceTime = &metav1.Time{sinceTime}
		podList := &corev1.PodList{}
		err := framework.Global.Client.List(context.TODO(), podList, &dynclient.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		podsClientset := framework.Global.KubeClient.CoreV1().Pods(namespace)

		for _, pod := range podList.Items {
			appLabel, hasAppLabel := pod.Labels["app"]
			if !hasAppLabel || appLabel != "kubernetes-nmstate" {
				continue
			}
			req := podsClientset.GetLogs(pod.Name, &podLogOpts)
			podLogs, err := req.Stream()
			if err != nil {
				io.WriteString(GinkgoWriter, fmt.Sprintf("error in opening stream: %v\n", err))
				continue
			}
			defer podLogs.Close()
			rawLogs, err := ioutil.ReadAll(podLogs)
			if err != nil {
				io.WriteString(GinkgoWriter, fmt.Sprintf("error reading kubernetes-nmstate logs: %v\n", err))
				continue
			}
			formattedLogs := strings.Replace(string(rawLogs), "\\n", "\n", -1)
			io.WriteString(writer, formattedLogs)
		}
	})
	return nil
}

func (r *KubernetesNMStateReporter) OpenTestLogFile(logType string, testName string, cb func(f io.ReadWriteCloser)) {
	name := fmt.Sprintf("%s/%s_%s.log", r.artifactsDir, testName, logType)
	fi, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := fi.Close(); err != nil {
			fmt.Println(err)
			return
		}
	}()
	cb(fi)
}
