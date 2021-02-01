package nmstatectl

import (
	"bytes"
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func ShowAtNode(config *rest.Config, nodeName string) (string, error) {
	cmd := []string{
		"nmstatectl",
		"show",
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	//TODO: Don't hardcode namespace
	//TODO: Use controller-runtime client ?
	handlerPodList, err := client.CoreV1().Pods("nmstate").List(context.Background(), metav1.ListOptions{
		LabelSelector: "component=kubernetes-nmstate-handler",
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		return "", err
	}
	if len(handlerPodList.Items) == 0 {
		return "", fmt.Errorf("No handler running at %s", nodeName)
	}
	//TODO: Don't hardcode namespace
	handlerPod := handlerPodList.Items[0]
	req := client.CoreV1().RESTClient().Post().Resource("pods").Name(handlerPod.Name).
		Namespace(handlerPod.Namespace).SubResource("exec")
	option := &v1.PodExecOptions{
		Command: cmd,
		Stdin:   false,
		Stdout:  true,
		Stderr:  true,
		TTY:     false,
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", err
	}
	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}
