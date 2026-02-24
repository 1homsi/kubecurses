package k8s

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// InteractiveExecCommand returns the default shell used for interactive exec sessions.
func InteractiveExecCommand() []string {
	return []string{"/bin/sh"}
}

// ExecInteractive opens a full interactive TTY session in the given pod/container.
// stdin is connected to the remote shell; all output (stdout+stderr) is written to
// stdout — Kubernetes merges stderr into the PTY when TTY=true.
// resizeQueue forwards terminal resize events so the remote PTY tracks the local window.
//
// The function blocks until the remote command exits, the context is cancelled, or
// a transport error occurs.
func ExecInteractive(
	ctx context.Context,
	cs *kubernetes.Clientset,
	config *rest.Config,
	namespace, pod, container string,
	command []string,
	stdin io.Reader,
	stdout io.Writer,
	resizeQueue remotecommand.TerminalSizeQueue,
) error {
	req := cs.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    false, // TTY mode: the PTY merges stderr into stdout
			TTY:       true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("create executor: %w", err)
	}

	return executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Tty:               true,
		TerminalSizeQueue: resizeQueue,
	})
}
