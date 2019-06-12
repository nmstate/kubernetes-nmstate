package handler

import (
	"bytes"
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Handler struct {
	pod corev1.Pod
}

func New(controllerClient client.Client, nodeName string) (Handler, error) {
	handler := Handler{}
	// Fetch the handler pod at this node
	podList := corev1.PodList{}
	listOptions := &client.ListOptions{}
	// We have to look by label, since is the  only cached stuff
	// matching by field does not work
	listOptions.MatchingLabels(map[string]string{"nmstate.io": "nmstate-handler"})
	err := controllerClient.List(context.TODO(), listOptions, &podList)
	if err != nil {
		return handler, fmt.Errorf("Error listing nmstate-handler pods: %v", err)
	}
	if len(podList.Items) == 0 {
		return handler, fmt.Errorf("No nmstate-handler pods at cluster")
	}
	for _, pod := range podList.Items {
		if pod.Spec.NodeName == nodeName {
			handler.pod = pod
			return handler, nil
		}
	}
	// Just select the first one in case that more exists
	return handler, fmt.Errorf("No nmstate handlers at %s", nodeName)
}

func (h Handler) Nmstatectl(arguments string) (string, error) {

	kubeConfig, err := config.GetConfig()
	if err != nil {
		return "", fmt.Errorf("Impossible to get k8s config: %v", err)
	}

	gv := corev1.SchemeGroupVersion
	kubeConfig.GroupVersion = &gv
	kubeConfig.APIPath = "/api"
	kubeConfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	restClient, err := restclient.RESTClientFor(kubeConfig)
	if err != nil {
		return "", fmt.Errorf("Failure creating new k8s client: %v", err)
	}

	containerName := h.pod.Spec.Containers[0].Name
	req := restClient.Post().Resource("pods").
		Name(h.pod.Name).
		Namespace(h.pod.Namespace).
		SubResource("exec").
		Param("container", containerName)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: containerName,
		Command:   []string{"/bin/bash", "-c", "nmstatectl " + arguments},
		Stdin:     false,
		Stderr:    true,
		Stdout:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	exec, err := remotecommand.NewSPDYExecutor(kubeConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("Error at executor init %v", err)
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("stdout: %s, stderr: %s, err: %v", stdout.String(), stderr.String(), err)
	}
	fmt.Printf("stdout: %s, stderr: %s, err: %v\n", stdout.String(), stderr.String(), err)

	return stdout.String(), nil
}
