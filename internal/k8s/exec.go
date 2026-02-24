package k8s

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// DefaultExecCommand returns the shell invocation used by the exec MVP.
// It runs a brief info command non-interactively so the output overlay
// is immediately useful without requiring a full PTY.
func DefaultExecCommand() []string {
	return []string{"/bin/sh", "-lc", "id && uname -a && (ls -la || true)"}
}

// ExecCommand runs command in the given pod/container and streams each output
// line (stdout + stderr interleaved) to lines. It returns when the command
// exits, the context is cancelled, or a transport error occurs.
//
// The function is intentionally synchronous so the caller controls the
// goroutine lifetime via context cancel, matching the pattern used by
// StreamLogs.
//
// Architecture — extension points for full interactive shell mode:
//   - Set Stdin: stdinReader in StreamOptions (PodExecOptions.Stdin=true)
//     to wire raw key passthrough from the TUI.
//   - Set TTY: true in PodExecOptions + pass a TerminalSizeQueue to propagate
//     terminal resize events.
func ExecCommand(
	ctx context.Context,
	cs *kubernetes.Clientset,
	config *rest.Config,
	namespace, pod, container string,
	command []string,
	lines chan<- string,
) error {
	req := cs.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     false, // TODO: true for interactive mode
			Stdout:    true,
			Stderr:    true,
			TTY:       false, // TODO: true + TerminalSizeQueue for interactive mode
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("create executor: %w", err)
	}

	outR, outW := io.Pipe()
	errR, errW := io.Pipe()

	// Stream goroutine: drives the SPDY connection and closes both pipes when
	// done so the scanner goroutines receive EOF and exit cleanly.
	streamDone := make(chan error, 1)
	go func() {
		streamErr := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdout: outW,
			Stderr: errW,
			// TODO: Stdin:             stdinReader for interactive mode
			// TODO: TerminalSizeQueue: resizeQueue for resize propagation
		})
		outW.CloseWithError(streamErr)
		errW.CloseWithError(streamErr)
		streamDone <- streamErr
	}()

	// Forward stdout and stderr lines concurrently so that a slow consumer on
	// one pipe cannot deadlock the stream goroutine blocked writing to the other.
	var wg sync.WaitGroup
	scan := func(r *io.PipeReader) {
		defer wg.Done()
		defer r.Close() // unblock any pending write if we exit early via ctx
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			select {
			case lines <- sc.Text():
			case <-ctx.Done():
				return
			}
		}
	}
	wg.Add(2)
	go scan(outR)
	go scan(errR)

	wg.Wait()
	return <-streamDone
}
