package k8s

import (
	"context"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/1homsi/kubecurses/internal/model"
)

const updateChanCap = 32

// Watcher runs background goroutines that periodically fetch Kubernetes
// resources and send model.Update values on a shared channel.
type Watcher struct {
	cs        *kubernetes.Clientset
	namespace string
	updates   chan model.Update
	refresh   chan struct{}
}

// NewWatcher creates a Watcher. Call Start to begin polling.
func NewWatcher(cs *kubernetes.Clientset, namespace string) *Watcher {
	return &Watcher{
		cs:        cs,
		namespace: namespace,
		updates:   make(chan model.Update, updateChanCap),
		refresh:   make(chan struct{}, 1),
	}
}

// Updates returns the channel on which model.Update values are sent.
func (w *Watcher) Updates() <-chan model.Update {
	return w.updates
}

// Clientset returns the underlying Kubernetes clientset.
func (w *Watcher) Clientset() *kubernetes.Clientset {
	return w.cs
}

// TriggerRefresh signals all watcher goroutines to re-fetch immediately.
func (w *Watcher) TriggerRefresh() {
	select {
	case w.refresh <- struct{}{}:
	default:
	}
}

// Start launches one goroutine per resource type.
func (w *Watcher) Start(ctx context.Context, interval time.Duration) {
	go w.watchPods(ctx, interval)
	go w.watchNodes(ctx, interval)
	go w.watchNamespaces(ctx, interval)
	go w.watchDeployments(ctx, interval)
}

func (w *Watcher) send(ctx context.Context, u model.Update) {
	select {
	case w.updates <- u:
	case <-ctx.Done():
	}
}

func (w *Watcher) watchPods(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	w.fetchAndSendPods(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.fetchAndSendPods(ctx)
		case <-w.refresh:
			w.fetchAndSendPods(ctx)
		}
	}
}

func (w *Watcher) fetchAndSendPods(ctx context.Context) {
	pods, err := FetchPods(ctx, w.cs, w.namespace)
	if err != nil {
		w.send(ctx, model.Update{Kind: model.UpdatePods, Err: err})
		return
	}
	// Best-effort: enrich Pending pods with the last FailedScheduling reason.
	if reasons, rerr := FetchPendingReasons(ctx, w.cs); rerr == nil {
		for i := range pods {
			if pods[i].Status == "Pending" {
				pods[i].PendingReason = reasons[pods[i].Name]
			}
		}
	}
	w.send(ctx, model.Update{Kind: model.UpdatePods, Pods: pods})
}

func (w *Watcher) watchNodes(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	w.fetchAndSendNodes(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.fetchAndSendNodes(ctx)
		case <-w.refresh:
			w.fetchAndSendNodes(ctx)
		}
	}
}

func (w *Watcher) fetchAndSendNodes(ctx context.Context) {
	nodes, err := FetchNodes(ctx, w.cs)
	if err != nil {
		w.send(ctx, model.Update{Kind: model.UpdateNodes, Err: err})
		return
	}
	// Best-effort: merge metrics-server data if available.
	if metrics, _ := FetchNodeMetrics(ctx, w.cs); metrics != nil {
		for i := range nodes {
			if m, ok := metrics[nodes[i].Name]; ok {
				nodes[i].UsedCPUm = m.cpuM
				nodes[i].UsedMemMi = m.memMi
				nodes[i].MetricsOK = true
			}
		}
	}
	w.send(ctx, model.Update{Kind: model.UpdateNodes, Nodes: nodes})
}

func (w *Watcher) watchNamespaces(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	w.fetchAndSendNamespaces(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.fetchAndSendNamespaces(ctx)
		case <-w.refresh:
			w.fetchAndSendNamespaces(ctx)
		}
	}
}

func (w *Watcher) fetchAndSendNamespaces(ctx context.Context) {
	nss, err := FetchNamespaces(ctx, w.cs)
	w.send(ctx, model.Update{Kind: model.UpdateNamespaces, Namespaces: nss, Err: err})
}

func (w *Watcher) watchDeployments(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	w.fetchAndSendDeployments(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.fetchAndSendDeployments(ctx)
		case <-w.refresh:
			w.fetchAndSendDeployments(ctx)
		}
	}
}

func (w *Watcher) fetchAndSendDeployments(ctx context.Context) {
	deps, err := FetchDeployments(ctx, w.cs, w.namespace)
	w.send(ctx, model.Update{Kind: model.UpdateDeployments, Deployments: deps, Err: err})
}
