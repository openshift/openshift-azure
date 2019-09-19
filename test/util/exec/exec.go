package exec

import (
	"bytes"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/openshift/openshift-azure/test/clients/openshift"
)

func RunCommandInPod(client *openshift.Client, pod *corev1.Pod, command string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	req := client.CoreV1.RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command:   []string{"/bin/sh", "-c", command},
			Container: pod.Spec.Containers[0].Name,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	executor, err := client.CommandExecutor(http.MethodPost, req.URL())
	if err != nil {
		return "", "", err
	}

	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return "", "", err
	}

	return stdout.String(), stderr.String(), nil
}
