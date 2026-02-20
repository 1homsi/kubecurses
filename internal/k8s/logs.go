package k8s

import (
	"bufio"
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// StreamLogs tails and follows the logs of a pod/container, sending each line
// to the provided channel. Returns when the context is cancelled or the stream
// closes. The caller must close the context to stop streaming.
func StreamLogs(ctx context.Context, cs *kubernetes.Clientset, namespace, pod, container string, tailLines int64, lines chan<- string) {
	opts := &corev1.PodLogOptions{
		Container: container,
		Follow:    true,
		TailLines: &tailLines,
	}
	req := cs.CoreV1().Pods(namespace).GetLogs(pod, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return // pod not running, not found, etc.
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		select {
		case lines <- scanner.Text():
		case <-ctx.Done():
			return
		}
	}
}
