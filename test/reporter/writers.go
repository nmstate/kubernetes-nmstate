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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
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
		if err != nil {
			writeMessage(writer, banner("failed reporting NetworkManager logs at %s: %v"), node, err)
		} else {
			writeMessage(writer, banner("Journalctl for NetworkManager on node %s"), node)
			writeMessage(writer, banner("\n %s"), output)
			writeMessage(writer, banner("Done NetworkManager logs on node %s"), node)
		}
	}
}

func networkManagerLogsWriter(nodes []string, sinceTime time.Time) func(io.Writer) {
	return func(w io.Writer) {
		writeNetworkManagerLogs(w, nodes, sinceTime)
	}
}

func writePodsLogs(writer io.Writer, namespace string, sinceTime time.Time) {
	podList := &corev1.PodList{}
	err := testenv.Client.List(context.TODO(), podList, &dynclient.ListOptions{})
	Expect(err).ToNot(HaveOccurred())
	podsClientset := testenv.KubeClient.CoreV1().Pods(namespace)

	for podIndex := range podList.Items {
		pod := &podList.Items[podIndex]
		appLabel, hasAppLabel := pod.Labels["app"]
		if !hasAppLabel || appLabel != "kubernetes-nmstate" {
			continue
		}
		for containerIndex := range pod.Spec.Containers {
			containerName := pod.Spec.Containers[containerIndex].Name
			podLogOpts := corev1.PodLogOptions{
				SinceTime: &metav1.Time{Time: sinceTime},
				Container: containerName,
			}
			podLogOpts.SinceTime = &metav1.Time{Time: sinceTime}
			req := podsClientset.GetLogs(pod.Name, &podLogOpts)
			podLogs, err := req.Stream(context.TODO())
			if err != nil {
				io.WriteString(GinkgoWriter, fmt.Sprintf("error in opening stream: %v\n", err))
				continue
			}
			defer podLogs.Close()
			rawLogs, err := io.ReadAll(podLogs)
			if err != nil {
				io.WriteString(GinkgoWriter, fmt.Sprintf("error reading kubernetes-nmstate logs: %v\n", err))
				continue
			}
			formattedLogs := strings.ReplaceAll(string(rawLogs), "\\n", "\n")
			io.WriteString(writer, formattedLogs)
		}
	}
}

func podLogsWriter(namespace string, sinceTime time.Time) func(io.Writer) {
	return func(w io.Writer) {
		writePodsLogs(w, namespace, sinceTime)
	}
}

func writeJournalctl(writer io.Writer, nodes []string, sinceTime time.Time) {
	for _, node := range nodes {
		output, err := runner.RunQuietAtNode(node, "sudo", "journalctl", "--no-pager",
			"--since", fmt.Sprintf("'%ds ago'", 10+int(time.Since(sinceTime).Seconds())))
		if err != nil {
			writeMessage(writer, banner("failed collecting journalctl at %s: %v"), node, err)
		} else {
			writeMessage(writer, banner("Journalctl on node %s"), node)
			writeMessage(writer, "\n%s", output)
			writeMessage(writer, banner("Done journalctl on node %s"), node)
		}
	}
}

func journalctlWriter(nodes []string, sinceTime time.Time) func(io.Writer) {
	return func(w io.Writer) {
		writeJournalctl(w, nodes, sinceTime)
	}
}

func writeDmesg(writer io.Writer, nodes []string) {
	for _, node := range nodes {
		output, err := runner.RunQuietAtNode(node, "sudo", "dmesg")
		if err != nil {
			writeMessage(writer, banner("failed collecting dmesg at %s: %v"), node, err)
		} else {
			writeMessage(writer, banner("dmesg on node %s"), node)
			writeMessage(writer, "\n%s", output)
			writeMessage(writer, banner("Done dmesg on node %s"), node)
		}
	}
}

func dmesgWriter(nodes []string) func(io.Writer) {
	return func(w io.Writer) {
		writeDmesg(w, nodes)
	}
}

func writeKubeletLogs(writer io.Writer, nodes []string, sinceTime time.Time) {
	for _, node := range nodes {
		output, err := runner.RunQuietAtNode(node, "sudo", "journalctl", "-u", "kubelet", "--no-pager",
			"--since", fmt.Sprintf("'%ds ago'", 10+int(time.Since(sinceTime).Seconds())))
		if err != nil {
			writeMessage(writer, banner("failed collecting kubelet logs at %s: %v"), node, err)
		} else {
			writeMessage(writer, banner("Kubelet logs on node %s"), node)
			writeMessage(writer, "\n%s", output)
			writeMessage(writer, banner("Done kubelet logs on node %s"), node)
		}
	}
}

func kubeletLogsWriter(nodes []string, sinceTime time.Time) func(io.Writer) {
	return func(w io.Writer) {
		writeKubeletLogs(w, nodes, sinceTime)
	}
}

func writeNamespaceEvents(writer io.Writer, namespace string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	eventsList := &corev1.EventList{}
	err := testenv.Client.List(ctx, eventsList, &dynclient.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		writeMessage(writer, banner("failed listing events in namespace %s: %v"), namespace, err)
		return
	}
	writeMessage(writer, banner("Events in namespace %s"), namespace)
	for i := range eventsList.Items {
		event := &eventsList.Items[i]
		writeMessage(writer, "%s\t%s\t%s/%s\t%s\t%s\n",
			event.LastTimestamp.Format(time.RFC3339),
			event.Type, event.InvolvedObject.Kind, event.InvolvedObject.Name,
			event.Reason, event.Message)
	}
	writeMessage(writer, banner("Done events in namespace %s"), namespace)
}

func writeClusterEvents(writer io.Writer, namespace string) {
	writeNamespaceEvents(writer, namespace)
	writeNamespaceEvents(writer, "default")
}

func clusterEventsWriter(namespace string) func(io.Writer) {
	return func(w io.Writer) {
		writeClusterEvents(w, namespace)
	}
}

func writeControlPlaneLogs(writer io.Writer, sinceTime time.Time) {
	components := []struct {
		label string
		name  string
	}{
		{label: "component=kube-apiserver", name: "kube-apiserver"},
		{label: "component=etcd", name: "etcd"},
	}

	podsClientset := testenv.KubeClient.CoreV1().Pods("kube-system")

	for _, comp := range components {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		podList := &corev1.PodList{}
		err := testenv.Client.List(ctx, podList, &dynclient.ListOptions{
			Namespace: "kube-system",
		}, dynclient.MatchingLabels(parseLabel(comp.label)))
		if err != nil {
			cancel()
			writeMessage(writer, banner("failed listing %s pods: %v"), comp.name, err)
			continue
		}

		for podIndex := range podList.Items {
			pod := &podList.Items[podIndex]
			for containerIndex := range pod.Spec.Containers {
				containerName := pod.Spec.Containers[containerIndex].Name
				podLogOpts := corev1.PodLogOptions{
					SinceTime: &metav1.Time{Time: sinceTime},
					Container: containerName,
				}
				req := podsClientset.GetLogs(pod.Name, &podLogOpts)
				podLogs, err := req.Stream(ctx)
				if err != nil {
					writeMessage(writer, banner("failed getting %s logs from pod %s: %v"), comp.name, pod.Name, err)
					continue
				}
				rawLogs, err := io.ReadAll(podLogs)
				podLogs.Close()
				if err != nil {
					writeMessage(writer, banner("failed reading %s logs from pod %s: %v"), comp.name, pod.Name, err)
					continue
				}
				writeMessage(writer, banner("%s logs from pod %s"), comp.name, pod.Name)
				writeString(writer, string(rawLogs))
				writeMessage(writer, banner("Done %s logs from pod %s"), comp.name, pod.Name)
			}
		}
		cancel()
	}
}

func parseLabel(label string) map[string]string {
	parts := strings.SplitN(label, "=", 2)
	if len(parts) == 2 && parts[0] != "" {
		return map[string]string{parts[0]: parts[1]}
	}
	return map[string]string{}
}

func controlPlaneLogsWriter(sinceTime time.Time) func(io.Writer) {
	return func(w io.Writer) {
		writeControlPlaneLogs(w, sinceTime)
	}
}

func banner(message string) string {
	// Not use Sprintf so we don't have to escape expansions
	return "\n" + separator + " " + message + " " + separator + "\n"
}

func writeString(writer io.Writer, message string) {
	writer.Write([]byte(message))
}

func writeMessage(writer io.Writer, message string, args ...any) {
	formattedMessage := message
	if len(args) > 0 {
		formattedMessage = fmt.Sprintf(formattedMessage, args...)
	}
	writeString(writer, formattedMessage)
}
