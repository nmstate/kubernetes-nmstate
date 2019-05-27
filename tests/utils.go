package nmstate_tests

import (
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate.io/v1"
)

func findInterfaceInfo(name string, interfaces []nmstatev1.InterfaceInfo) (nmstatev1.InterfaceInfo, bool) {
	for _, iface := range interfaces {
		if iface.Name == name {
			return iface, true
		}
	}
	return nmstatev1.InterfaceInfo{}, false
}

func findInterfaceSpec(name string, interfaces []nmstatev1.InterfaceSpec) (nmstatev1.InterfaceSpec, bool) {
	for _, iface := range interfaces {
		if iface.Name == name {
			return iface, true
		}
	}
	return nmstatev1.InterfaceSpec{}, false
}

func isReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady &&
			condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func waitPodsCleanup() error {
	return wait.Poll(10, 30*time.Second, func() (bool, error) {
		pods, err := nmstatePodsClient.List(metav1.ListOptions{})
		if err != nil {
			return true, fmt.Errorf("error listing pods: %v", err)
		}
		return len(pods.Items) == 0, nil
	})
}

func waitPodsReady() error {
	return wait.Poll(10, 30*time.Second, func() (bool, error) {
		pods, err := nmstatePodsClient.List(metav1.ListOptions{})
		if err != nil {
			return true, fmt.Errorf("error listing pods: %v", err)
		}
		if len(pods.Items) == 0 {
			return false, nil
		}
		for _, pod := range pods.Items {
			if !isReady(pod) {
				return false, nil
			}
		}
		return true, nil
	})
}

func writePodsLogs(writer io.Writer) error {
	podLogOpts := corev1.PodLogOptions{}
	pods, err := nmstatePodsClient.List(metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing pods: %v", err)
	}
	for _, pod := range pods.Items {
		req := nmstatePodsClient.GetLogs(pod.Name, &podLogOpts)
		podLogs, err := req.Stream()
		if err != nil {
			io.WriteString(writer, fmt.Sprintf("error in opening stream: %v\n", err))
			continue
		}
		defer podLogs.Close()
		_, err = io.Copy(writer, podLogs)
		if err != nil {
			io.WriteString(writer, fmt.Sprintf("error in copy information from podLogs to buf: %v\n", err))
			continue
		}

	}
	return nil
}
