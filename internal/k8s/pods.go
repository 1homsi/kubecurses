package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/1homsi/kubecurses/internal/model"
)

// FetchPods lists pods in the given namespace ("" = all namespaces).
func FetchPods(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]model.Pod, error) {
	list, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	now := time.Now()
	pods := make([]model.Pod, 0, len(list.Items))
	for _, p := range list.Items {
		ready, total := 0, len(p.Status.ContainerStatuses)
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
		}
		var restarts int32
		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}
		age := now.Sub(p.CreationTimestamp.Time).Truncate(time.Second)
		pods = append(pods, model.Pod{
			Namespace: p.Namespace,
			Name:      p.Name,
			Ready:     fmt.Sprintf("%d/%d", ready, total),
			Status:    string(p.Status.Phase),
			Restarts:  restarts,
			Age:       age,
			Node:      p.Spec.NodeName,
		})
	}
	return pods, nil
}
