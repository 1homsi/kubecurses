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

// convertNamespace converts a corev1.Namespace to a model.Namespace using the given time as "now".
func convertNamespace(ns corev1.Namespace, now time.Time) model.Namespace {
	return model.Namespace{
		Name:   ns.Name,
		Status: string(ns.Status.Phase),
		Age:    now.Sub(ns.CreationTimestamp.Time).Truncate(time.Second),
	}
}

// FetchNamespaces lists all namespaces in the cluster.
func FetchNamespaces(ctx context.Context, cs *kubernetes.Clientset) ([]model.Namespace, error) {
	list, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	now := time.Now()
	nss := make([]model.Namespace, 0, len(list.Items))
	for _, ns := range list.Items {
		nss = append(nss, convertNamespace(ns, now))
	}
	return nss, nil
}
