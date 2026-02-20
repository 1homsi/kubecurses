package k8s

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/1homsi/kubecurses/internal/model"
)

// convertDeployment converts an appsv1.Deployment to a model.Deployment using the given time as "now".
func convertDeployment(d appsv1.Deployment, now time.Time) model.Deployment {
	desired := d.Spec.Replicas
	var desiredCount int32
	if desired != nil {
		desiredCount = *desired
	}
	return model.Deployment{
		Namespace: d.Namespace,
		Name:      d.Name,
		Ready:     fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, desiredCount),
		UpToDate:  d.Status.UpdatedReplicas,
		Available: d.Status.AvailableReplicas,
		Age:       now.Sub(d.CreationTimestamp.Time).Truncate(time.Second),
	}
}

// FetchDeployments lists deployments in the given namespace ("" = all namespaces).
func FetchDeployments(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]model.Deployment, error) {
	list, err := cs.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}

	now := time.Now()
	deps := make([]model.Deployment, 0, len(list.Items))
	for _, d := range list.Items {
		deps = append(deps, convertDeployment(d, now))
	}
	return deps, nil
}
