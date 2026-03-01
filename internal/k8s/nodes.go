package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/1homsi/kubecurses/internal/model"
)

func convertNode(n corev1.Node, now time.Time) model.Node {
	status := "NotReady"
	for _, c := range n.Status.Conditions {
		if c.Type == "Ready" && c.Status == "True" {
			status = "Ready"
			break
		}
	}

	var roles []string
	for label := range n.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			roles = append(roles, role)
		}
	}
	rolesStr := strings.Join(roles, ",")
	if rolesStr == "" {
		rolesStr = "<none>"
	}

	cpuQ := n.Status.Allocatable["cpu"]
	memQ := n.Status.Allocatable["memory"]
	podsQ := n.Status.Allocatable["pods"]

	taints := make([]string, 0, len(n.Spec.Taints))
	for _, t := range n.Spec.Taints {
		s := t.Key
		if t.Value != "" {
			s += "=" + t.Value
		}
		if t.Effect != "" {
			s += ":" + string(t.Effect)
		}
		taints = append(taints, s)
	}

	return model.Node{
		Name:       n.Name,
		Status:     status,
		Roles:      rolesStr,
		Age:        now.Sub(n.CreationTimestamp.Time).Truncate(time.Second),
		Version:    n.Status.NodeInfo.KubeletVersion,
		AllocCPUm:  cpuQ.MilliValue(),
		AllocMemMi: memQ.Value() / (1024 * 1024),
		AllocPods:  int(podsQ.Value()),
		Taints:     taints,
	}
}

// FetchNodes lists all nodes in the cluster and populates allocatable resources.
func FetchNodes(ctx context.Context, cs *kubernetes.Clientset) ([]model.Node, error) {
	list, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	now := time.Now()
	nodes := make([]model.Node, 0, len(list.Items))
	for _, n := range list.Items {
		nodes = append(nodes, convertNode(n, now))
	}
	return nodes, nil
}
