package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
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
		var restarts int32
		containers := make([]model.Container, 0, total)
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
			restarts += cs.RestartCount

			cStatus := "Running"
			if cs.State.Waiting != nil {
				cStatus = cs.State.Waiting.Reason
				if cStatus == "" {
					cStatus = "Waiting"
				}
			} else if cs.State.Terminated != nil {
				cStatus = cs.State.Terminated.Reason
				if cStatus == "" {
					cStatus = "Terminated"
				}
			}
			containers = append(containers, model.Container{
				Name:     cs.Name,
				Ready:    cs.Ready,
				Restarts: cs.RestartCount,
				Status:   cStatus,
			})
		}
		age := now.Sub(p.CreationTimestamp.Time).Truncate(time.Second)
		pods = append(pods, model.Pod{
			Namespace:  p.Namespace,
			Name:       p.Name,
			Ready:      fmt.Sprintf("%d/%d", ready, total),
			Status:     podEffectiveStatus(p),
			Restarts:   restarts,
			Age:        age,
			Node:       p.Spec.NodeName,
			Containers: containers,
		})
	}
	return pods, nil
}

// podEffectiveStatus returns a human-readable status string that reflects real pod
// state beyond just Phase — detects Terminating, CrashLoopBackOff, OOMKilled, etc.
func podEffectiveStatus(p corev1.Pod) string {
	if p.DeletionTimestamp != nil {
		return "Terminating"
	}
	for _, cs := range p.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			switch cs.State.Waiting.Reason {
			case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull",
				"CreateContainerConfigError", "InvalidImageName":
				return cs.State.Waiting.Reason
			}
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason == "OOMKilled" {
			return "OOMKilled"
		}
	}
	return string(p.Status.Phase)
}
