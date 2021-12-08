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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	testenv "github.com/nmstate/kubernetes-nmstate/test/env"
	runner "github.com/nmstate/kubernetes-nmstate/test/runner"
)

const (
	separator = "*****"
)

func writeDeviceStatus(writer io.Writer, nodes []string) {
	writeMessage(writer, banner("Start printing device status")+"\n")

	for _, node := range nodes {
		output, err := runner.RunQuietAtNode(node, "/usr/bin/nmcli", "c", "s")
		Expect(err).ToNot(HaveOccurred())

		writeMessage(writer, banner("Connection status on node %s"), node)
		writeMessage(writer, "\n %s", output)
		writeMessage(writer, banner("Done Connection status on node %s "), node)

		output, err = runner.RunQuietAtNode(node, "/usr/bin/nmcli", "d", "s")
		Expect(err).ToNot(HaveOccurred())

		writeMessage(writer, banner("Device status on node %s"), node)
		writeMessage(writer, "\n %s", output)
		writeMessage(writer, banner("Done device status on node %s "), node)

		output, err = runner.RunQuietAtNode(node, "/usr/sbin/ip", "-4", "-o", "a")
		Expect(err).ToNot(HaveOccurred())

		writeMessage(writer, banner("Configured ipv4 ips on devices on node %s"), node)
		writeMessage(writer, "\n %s", output)
		writeMessage(writer, banner("Done ip status on node %s"), node)
	}
	writeMessage(writer, "Finished printing device status")
}

func writeNetworkManagerLogs(writer io.Writer, nodes []string, sinceTime time.Time) {
	for _, node := range nodes {
		output, err := runner.RunQuietAtNode(node, "sudo", "journalctl", "-u", "NetworkManager",
			"--since", fmt.Sprintf("'%ds ago'", 10+int(time.Since(sinceTime).Seconds())))
		Expect(err).ToNot(HaveOccurred())
		writeMessage(writer, banner("Journalctl for NetworkManager on node %s"), node)
		writeMessage(writer, banner("\n %s"), output)
		writeMessage(writer, banner("Done NetworkManager logs on node %s"), node)
	}
}

func networkManagerLogsWriter(nodes []string, sinceTime time.Time) func(io.Writer) {
	return func(w io.Writer) {
		writeNetworkManagerLogs(w, nodes, sinceTime)
	}
}

func writePodsLogs(writer io.Writer, namespace string, sinceTime time.Time) {
	podLogOpts := corev1.PodLogOptions{}
	podLogOpts.SinceTime = &metav1.Time{Time: sinceTime}
	podList := &corev1.PodList{}
	err := testenv.Client.List(context.TODO(), podList, &dynclient.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	podsClientset := testenv.KubeClient.CoreV1().Pods(namespace)

	for _, pod := range podList.Items {
		appLabel, hasAppLabel := pod.Labels["app"]
		if !hasAppLabel || appLabel != "kubernetes-nmstate" {
			continue
		}
		req := podsClientset.GetLogs(pod.Name, &podLogOpts)
		podLogs, err := req.Stream(context.TODO())
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
}

func podLogsWriter(namespace string, sinceTime time.Time) func(io.Writer) {
	return func(w io.Writer) {
		writePodsLogs(w, namespace, sinceTime)
	}
}

func banner(message string) string {
	// Not use Sprintf so we don't have to escape expansions
	return "\n" + separator + " " + message + " " + separator + "\n"
}

func writeString(writer io.Writer, message string) {
	writer.Write([]byte(message))
}

func writeMessage(writer io.Writer, message string, args ...string) {
	formattedMessage := message
	if len(args) > 0 {
		formattedMessage = fmt.Sprintf(formattedMessage, args)
	}
	writeString(writer, formattedMessage)
}
