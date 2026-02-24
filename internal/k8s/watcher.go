package k8s

import (
	"context"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/1homsi/kubecurses/internal/model"
)

const (
	updateChanCap    = 32
	pendingReasonsTTL = 30 * time.Second
)

// WatcherOptions controls how the Watcher fetches and delivers updates.
type WatcherOptions struct {
	Watch           bool          // true = informer mode, false = polling
	PollInterval    time.Duration // used in polling mode
	MetricsInterval time.Duration // how often to refresh metrics (informer mode)
	DisableMetrics  bool          // skip metrics-server calls entirely
	MaxPods         int           // cap pod list (0 = unlimited)
}

// Watcher runs background goroutines that fetch Kubernetes resources and send
// model.Update values on a shared channel. Supports both informer-based Watch
// mode and periodic polling mode.
type Watcher struct {
	cs        *kubernetes.Clientset
	namespace string
	opts      WatcherOptions
	updates   chan model.Update
	refresh   chan struct{}

	// pendingReasonsCache is keyed by pod name and refreshed at most once per
	// pendingReasonsTTL. Only the pod worker goroutine touches these fields.
	pendingReasonsCache   map[string]string
	pendingReasonsFetchAt time.Time

	// nodeBaseCache holds the last node list without metrics so that metrics
	// ticks can patch-in usage values without re-listing all nodes.
	// Protected by nodeBaseMu because the node-change worker and metrics
	// goroutine run concurrently.
	nodeBaseMu    sync.Mutex
	nodeBaseCache []model.Node
}

// NewWatcher creates a Watcher. Call Start to begin.
func NewWatcher(cs *kubernetes.Clientset, namespace string, opts WatcherOptions) *Watcher {
	return &Watcher{
		cs:        cs,
		namespace: namespace,
		opts:      opts,
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

// TriggerRefresh signals polling goroutines to re-fetch immediately.
// In Watch mode this is a no-op.
func (w *Watcher) TriggerRefresh() {
	if w.opts.Watch {
		return
	}
	select {
	case w.refresh <- struct{}{}:
	default:
	}
}

// Start launches the background goroutines.
func (w *Watcher) Start(ctx context.Context) {
	if w.opts.Watch {
		go w.startInformers(ctx)
	} else {
		w.startPolling(ctx)
	}
}

// ── informer path ─────────────────────────────────────────────────────────────

func (w *Watcher) startInformers(ctx context.Context) {
	factory := informers.NewSharedInformerFactoryWithOptions(
		w.cs,
		10*time.Minute,
		informers.WithNamespace(w.namespace),
	)

	// Register informers with the factory before starting it.
	podInformer := factory.Core().V1().Pods()
	nodeInformer := factory.Core().V1().Nodes()
	nsInformer := factory.Core().V1().Namespaces()
	depInformer := factory.Apps().V1().Deployments()

	// Coalescing trigger channels (capacity 1 — drops redundant signals).
	podCh := make(chan struct{}, 1)
	nodeCh := make(chan struct{}, 1)
	nsCh := make(chan struct{}, 1)
	depCh := make(chan struct{}, 1)

	trig := func(ch chan<- struct{}) {
		select {
		case ch <- struct{}{}:
		default:
		}
	}

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { trig(podCh) },
		UpdateFunc: func(_, _ interface{}) { trig(podCh) },
		DeleteFunc: func(_ interface{}) { trig(podCh) },
	})
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { trig(nodeCh) },
		UpdateFunc: func(_, _ interface{}) { trig(nodeCh) },
		DeleteFunc: func(_ interface{}) { trig(nodeCh) },
	})
	nsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { trig(nsCh) },
		UpdateFunc: func(_, _ interface{}) { trig(nsCh) },
		DeleteFunc: func(_ interface{}) { trig(nsCh) },
	})
	depInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { trig(depCh) },
		UpdateFunc: func(_, _ interface{}) { trig(depCh) },
		DeleteFunc: func(_ interface{}) { trig(depCh) },
	})

	factory.Start(ctx.Done())

	// Send each resource type as soon as its own cache is ready — nodes appear
	// immediately without waiting for the (much larger) pod cache to sync.
	synced := make(chan struct{})
	go func() {
		cache.WaitForCacheSync(ctx.Done(), nodeInformer.Informer().HasSynced)
		w.sendNodesFromFactory(ctx, factory, nil)
	}()
	go func() {
		cache.WaitForCacheSync(ctx.Done(), nsInformer.Informer().HasSynced)
		w.sendNamespacesFromFactory(ctx, factory)
	}()
	go func() {
		cache.WaitForCacheSync(ctx.Done(), depInformer.Informer().HasSynced)
		w.sendDeploymentsFromFactory(ctx, factory)
	}()
	go func() {
		// Pods last — largest cache; send when ready, then signal all-synced.
		cache.WaitForCacheSync(ctx.Done(), podInformer.Informer().HasSynced)
		w.sendPodsFromFactory(ctx, factory)
		close(synced)
	}()

	// Wait for all caches before starting the event-driven workers so we
	// don't race with the initial sends above.
	select {
	case <-ctx.Done():
		return
	case <-synced:
	}

	// Metrics goroutine — always polled even in Watch mode.
	if !w.opts.DisableMetrics && w.opts.MetricsInterval > 0 {
		go w.watchMetricsFactory(ctx, factory)
	}

	// Pod worker.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-podCh:
				w.sendPodsFromFactory(ctx, factory)
			}
		}
	}()

	// Node worker.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-nodeCh:
				w.sendNodesFromFactory(ctx, factory, nil)
			}
		}
	}()

	// Namespace worker.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-nsCh:
				w.sendNamespacesFromFactory(ctx, factory)
			}
		}
	}()

	// Deployment worker.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-depCh:
				w.sendDeploymentsFromFactory(ctx, factory)
			}
		}
	}()
}

func (w *Watcher) sendPodsFromFactory(ctx context.Context, factory informers.SharedInformerFactory) {
	items, err := factory.Core().V1().Pods().Lister().List(labels.Everything())
	if err != nil {
		w.send(ctx, model.Update{Kind: model.UpdatePods, Err: err})
		return
	}
	now := time.Now()
	pods := make([]model.Pod, 0, len(items))
	for _, p := range items {
		pods = append(pods, convertPod(*p, now))
	}
	if w.opts.MaxPods > 0 && len(pods) > w.opts.MaxPods {
		pods = pods[:w.opts.MaxPods]
	}
	w.applyPendingReasons(ctx, pods)
	w.send(ctx, model.Update{Kind: model.UpdatePods, Pods: pods})
}

// applyPendingReasons decorates Pending pods with their last FailedScheduling
// reason, fetching from the API at most once per pendingReasonsTTL. Skips the
// API call entirely when there are no Pending pods.
func (w *Watcher) applyPendingReasons(ctx context.Context, pods []model.Pod) {
	hasPending := false
	for i := range pods {
		if pods[i].Status == "Pending" {
			hasPending = true
			break
		}
	}
	if !hasPending {
		return
	}
	if w.pendingReasonsCache != nil && time.Since(w.pendingReasonsFetchAt) < pendingReasonsTTL {
		for i := range pods {
			if pods[i].Status == "Pending" {
				pods[i].PendingReason = w.pendingReasonsCache[pods[i].Name]
			}
		}
		return
	}
	if reasons, err := FetchPendingReasons(ctx, w.cs); err == nil {
		w.pendingReasonsCache = reasons
		w.pendingReasonsFetchAt = time.Now()
		for i := range pods {
			if pods[i].Status == "Pending" {
				pods[i].PendingReason = reasons[pods[i].Name]
			}
		}
	}
}

func (w *Watcher) sendNodesFromFactory(ctx context.Context, factory informers.SharedInformerFactory, metrics map[string]nodeMetrics) {
	var nodes []model.Node

	if metrics == nil {
		// Node change: re-list from informer and refresh the base cache.
		items, err := factory.Core().V1().Nodes().Lister().List(labels.Everything())
		if err != nil {
			w.send(ctx, model.Update{Kind: model.UpdateNodes, Err: err})
			return
		}
		now := time.Now()
		base := make([]model.Node, 0, len(items))
		for _, n := range items {
			base = append(base, convertNode(*n, now))
		}
		w.nodeBaseMu.Lock()
		w.nodeBaseCache = base
		w.nodeBaseMu.Unlock()
		nodes = base
	} else {
		// Metrics tick: clone the cached base and patch usage values only.
		w.nodeBaseMu.Lock()
		base := w.nodeBaseCache
		if len(base) > 0 {
			nodes = make([]model.Node, len(base))
			copy(nodes, base)
		}
		w.nodeBaseMu.Unlock()
		if nodes == nil {
			return // base not populated yet; skip this tick
		}
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

func (w *Watcher) sendNamespacesFromFactory(ctx context.Context, factory informers.SharedInformerFactory) {
	items, err := factory.Core().V1().Namespaces().Lister().List(labels.Everything())
	if err != nil {
		w.send(ctx, model.Update{Kind: model.UpdateNamespaces, Err: err})
		return
	}
	now := time.Now()
	nss := make([]model.Namespace, 0, len(items))
	for _, ns := range items {
		nss = append(nss, convertNamespace(*ns, now))
	}
	w.send(ctx, model.Update{Kind: model.UpdateNamespaces, Namespaces: nss})
}

func (w *Watcher) sendDeploymentsFromFactory(ctx context.Context, factory informers.SharedInformerFactory) {
	items, err := factory.Apps().V1().Deployments().Lister().List(labels.Everything())
	if err != nil {
		w.send(ctx, model.Update{Kind: model.UpdateDeployments, Err: err})
		return
	}
	now := time.Now()
	deps := make([]model.Deployment, 0, len(items))
	for _, d := range items {
		deps = append(deps, convertDeployment(*d, now))
	}
	w.send(ctx, model.Update{Kind: model.UpdateDeployments, Deployments: deps})
}

func (w *Watcher) watchMetricsFactory(ctx context.Context, factory informers.SharedInformerFactory) {
	ticker := time.NewTicker(w.opts.MetricsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics, _ := FetchNodeMetrics(ctx, w.cs)
			w.sendNodesFromFactory(ctx, factory, metrics)
		}
	}
}

// ── polling path ──────────────────────────────────────────────────────────────

func (w *Watcher) startPolling(ctx context.Context) {
	go w.watchPods(ctx)
	go w.watchNodes(ctx)
	go w.watchNamespaces(ctx)
	go w.watchDeployments(ctx)
}

func (w *Watcher) send(ctx context.Context, u model.Update) {
	select {
	case w.updates <- u:
	case <-ctx.Done():
	}
}

func (w *Watcher) watchPods(ctx context.Context) {
	ticker := time.NewTicker(w.opts.PollInterval)
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
	if w.opts.MaxPods > 0 && len(pods) > w.opts.MaxPods {
		pods = pods[:w.opts.MaxPods]
	}
	w.applyPendingReasons(ctx, pods)
	w.send(ctx, model.Update{Kind: model.UpdatePods, Pods: pods})
}

func (w *Watcher) watchNodes(ctx context.Context) {
	ticker := time.NewTicker(w.opts.PollInterval)
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
	if !w.opts.DisableMetrics {
		if metrics, _ := FetchNodeMetrics(ctx, w.cs); metrics != nil {
			for i := range nodes {
				if m, ok := metrics[nodes[i].Name]; ok {
					nodes[i].UsedCPUm = m.cpuM
					nodes[i].UsedMemMi = m.memMi
					nodes[i].MetricsOK = true
				}
			}
		}
	}
	w.send(ctx, model.Update{Kind: model.UpdateNodes, Nodes: nodes})
}

func (w *Watcher) watchNamespaces(ctx context.Context) {
	ticker := time.NewTicker(w.opts.PollInterval)
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

func (w *Watcher) watchDeployments(ctx context.Context) {
	ticker := time.NewTicker(w.opts.PollInterval)
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
