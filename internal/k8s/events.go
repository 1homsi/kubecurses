package k8s

import (
	"context"
	"fmt"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/1homsi/kubecurses/internal/model"
)

// FetchPendingReasons returns a map of namespace/name → most recent FailedScheduling
// event message. A single API call fetches all such events cluster-wide.
func FetchPendingReasons(ctx context.Context, cs *kubernetes.Clientset) (map[string]string, error) {
	list, err := cs.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: "reason=FailedScheduling",
	})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	type entry struct {
		msg string
		ts  int64
	}
	best := make(map[string]entry, len(list.Items))
	for _, e := range list.Items {
		name := e.InvolvedObject.Namespace + "/" + e.InvolvedObject.Name
		ts := e.LastTimestamp.Unix()
		if cur, ok := best[name]; !ok || ts > cur.ts {
			best[name] = entry{msg: e.Message, ts: ts}
		}
	}

	reasons := make(map[string]string, len(best))
	for k, v := range best {
		reasons[k] = v.msg
	}
	return reasons, nil
}

// FetchEvents lists all events in the given namespace ("" = all namespaces),
// sorted with Warning type first, then by age ascending (newest first).
func FetchEvents(ctx context.Context, cs *kubernetes.Clientset, namespace string) ([]model.Event, error) {
	list, err := cs.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	now := time.Now()
	events := make([]model.Event, 0, len(list.Items))
	for _, e := range list.Items {
		age := now.Sub(e.LastTimestamp.Time)
		if age < 0 {
			age = 0
		}
		events = append(events, model.Event{
			Namespace: e.Namespace,
			Kind:      e.InvolvedObject.Kind,
			Name:      e.InvolvedObject.Name,
			Reason:    e.Reason,
			Message:   e.Message,
			Count:     e.Count,
			Age:       age.Truncate(time.Second),
			Type:      e.Type,
		})
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].Type != events[j].Type {
			return events[i].Type == "Warning"
		}
		return events[i].Age < events[j].Age
	})
	return events, nil
}
