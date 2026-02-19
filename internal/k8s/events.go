package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// FetchPendingReasons returns a map of pod-name → most recent FailedScheduling
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
		ts  int64 // unix seconds of LastTimestamp
	}
	best := make(map[string]entry, len(list.Items))
	for _, e := range list.Items {
		name := e.InvolvedObject.Name
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
