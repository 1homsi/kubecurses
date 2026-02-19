package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/1homsi/kubecurses/internal/model"
)

// FetchDeployments lists deployments in the given namespace ("" = all namespaces).
func FetchDeployments(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]model.Deployment, error) {
	list, err := cs.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}

	now := time.Now()
	deps := make([]model.Deployment, 0, len(list.Items))
	for _, d := range list.Items {
		desired := d.Spec.Replicas
		var desiredCount int32
		if desired != nil {
			desiredCount = *desired
		}
		ready := d.Status.ReadyReplicas
		age := now.Sub(d.CreationTimestamp.Time).Truncate(time.Second)
		deps = append(deps, model.Deployment{
			Namespace: d.Namespace,
			Name:      d.Name,
			Ready:     fmt.Sprintf("%d/%d", ready, desiredCount),
			UpToDate:  d.Status.UpdatedReplicas,
			Available: d.Status.AvailableReplicas,
			Age:       age,
		})
	}
	return deps, nil
}
