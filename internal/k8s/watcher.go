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
// The main goroutine should read from this channel.
func (w *Watcher) Updates() <-chan model.Update {
	return w.updates
}

// TriggerRefresh sends a signal to all watcher goroutines requesting an
// immediate re-fetch. Non-blocking: if the channel is already full the
// signal is dropped (a refresh is already pending).
func (w *Watcher) TriggerRefresh() {
	select {
	case w.refresh <- struct{}{}:
	default:
	}
}

// Start launches one goroutine per resource type. ctx cancellation stops all
// goroutines. interval controls the polling frequency.
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
	w.send(ctx, model.Update{Kind: model.UpdatePods, Pods: pods, Err: err})
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
	w.send(ctx, model.Update{Kind: model.UpdateNodes, Nodes: nodes, Err: err})
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
