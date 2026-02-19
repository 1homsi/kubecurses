package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/1homsi/kubecurses/internal/model"
)

// FetchNamespaces lists all namespaces in the cluster.
func FetchNamespaces(ctx context.Context, cs *kubernetes.Clientset) ([]model.Namespace, error) {
	list, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	now := time.Now()
	nss := make([]model.Namespace, 0, len(list.Items))
	for _, ns := range list.Items {
		age := now.Sub(ns.CreationTimestamp.Time).Truncate(time.Second)
		nss = append(nss, model.Namespace{
			Name:   ns.Name,
			Status: string(ns.Status.Phase),
			Age:    age,
		})
	}
	return nss, nil
}
