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

func convertPod(p corev1.Pod, now time.Time) model.Pod {
	specByName := make(map[string]corev1.Container, len(p.Spec.Containers))
	for _, c := range p.Spec.Containers {
		specByName[c.Name] = c
	}

	ready, total := 0, len(p.Status.ContainerStatuses)
	var restarts int32
	containers := make([]model.Container, 0, total)
	for _, cs := range p.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
		restarts += cs.RestartCount

		cStatus := "Running"
		var cMessage string
		if cs.State.Waiting != nil {
			cStatus = cs.State.Waiting.Reason
			if cStatus == "" {
				cStatus = "Waiting"
			}
			if cs.State.Waiting.Message != "" {
				cMessage = cs.State.Waiting.Message
				if len([]rune(cMessage)) > 120 {
					cMessage = string([]rune(cMessage)[:120])
				}
			}
		} else if cs.State.Terminated != nil {
			cStatus = cs.State.Terminated.Reason
			if cStatus == "" {
				cStatus = "Terminated"
			}
			if cs.State.Terminated.Message != "" {
				cMessage = cs.State.Terminated.Message
				if len([]rune(cMessage)) > 120 {
					cMessage = string([]rune(cMessage)[:120])
				}
			}
		}

		c := model.Container{
			Name:     cs.Name,
			Ready:    cs.Ready,
			Restarts: cs.RestartCount,
			Status:   cStatus,
			Message:  cMessage,
		}
		if spec, ok := specByName[cs.Name]; ok {
			c.Image = spec.Image
			if r, ok2 := spec.Resources.Requests[corev1.ResourceCPU]; ok2 {
				c.CPURequestM = r.MilliValue()
			}
			if r, ok2 := spec.Resources.Limits[corev1.ResourceCPU]; ok2 {
				c.CPULimitM = r.MilliValue()
			}
			if r, ok2 := spec.Resources.Requests[corev1.ResourceMemory]; ok2 {
				c.MemRequestMi = r.Value() / (1024 * 1024)
			}
			if r, ok2 := spec.Resources.Limits[corev1.ResourceMemory]; ok2 {
				c.MemLimitMi = r.Value() / (1024 * 1024)
			}
		}
		containers = append(containers, c)
	}
	return model.Pod{
		Namespace:  p.Namespace,
		Name:       p.Name,
		Ready:      fmt.Sprintf("%d/%d", ready, total),
		Status:     podEffectiveStatus(p),
		Restarts:   restarts,
		Age:        now.Sub(p.CreationTimestamp.Time).Truncate(time.Second),
		Node:       p.Spec.NodeName,
		Containers: containers,
	}
}

// FetchPods lists pods in the given namespace ("" = all namespaces).
func FetchPods(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]model.Pod, error) {
	list, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}

	now := time.Now()
	pods := make([]model.Pod, 0, len(list.Items))
	for _, p := range list.Items {
		pods = append(pods, convertPod(p, now))
	}
	return pods, nil
}

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
