// Package cmd handles CLI flag parsing and wires up the application.
package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/klog/v2"

	"github.com/1homsi/kubecurses/internal/app"
	"github.com/1homsi/kubecurses/internal/k8s"
	"github.com/1homsi/kubecurses/internal/ui"
)

// Execute parses flags and runs the application.
func Execute(version, commit, buildDate string) {
	// Silence klog (used by client-go / informers) so its messages
	// don't bleed through the Bubble Tea screen and corrupt the TUI.
	klog.InitFlags(nil)
	_ = flag.Set("logtostderr", "false")
	_ = flag.Set("alsologtostderr", "false")
	_ = flag.Set("stderrthreshold", "FATAL")
	_ = flag.Set("v", "0")
	klog.SetOutput(io.Discard)
	klog.LogToStderr(false)

	cfg := app.DefaultConfig()

	showVersion := flag.Bool("version", false, "print version and exit")
	theme := flag.String("theme", "auto", "color theme: auto, dark, or light")
	flag.StringVar(&cfg.Kubeconfig, "kubeconfig", "", "path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	flag.StringVar(&cfg.Context, "context", "", "kubernetes context to use")
	flag.StringVar(&cfg.Namespace, "namespace", "", "namespace to filter (default: all namespaces)")
	flag.DurationVar(&cfg.PollInterval, "interval", 10*time.Second, "polling interval for resource refresh (used when --watch=false)")
	flag.BoolVar(&cfg.Watch, "watch", true, "use informer-based live updates instead of polling")
	flag.DurationVar(&cfg.MetricsInterval, "metrics-interval", 30*time.Second, "how often to refresh metrics-server data")
	flag.BoolVar(&cfg.EnableMetrics, "enable-metrics", false, "enable metrics-server integration")
	flag.DurationVar(&cfg.RequestTimeout, "request-timeout", 30*time.Second, "timeout for Kubernetes API requests")
	var kubeAPIQPS float64 = 20
	flag.Float64Var(&kubeAPIQPS, "kube-api-qps", 20, "maximum QPS to the Kubernetes API server")
	flag.IntVar(&cfg.KubeAPIBurst, "kube-api-burst", 40, "maximum burst for throttle to the Kubernetes API server")
	flag.IntVar(&cfg.MaxPods, "max-pods", 0, "maximum number of pods to display (0 = unlimited)")
	flag.BoolVar(&cfg.NoIcons, "no-icons", false, "replace icons with plain-text fallbacks")
	flag.Int64Var(&cfg.LogTailLines, "log-lines", 200, "number of log lines to tail when opening pod logs")
	flag.Parse()

	cfg.KubeAPIQPS = float32(kubeAPIQPS)

	ui.InitTheme(*theme)

	if *showVersion {
		fmt.Printf("kubecurses %s\ncommit:    %s\nbuilt:     %s\n", version, commit, buildDate)
		os.Exit(0)
	}

	// Context picker: run a mini tea.Program when no --context flag was given.
	if cfg.Context == "" {
		contexts, current, err := k8s.ListContexts(cfg.Kubeconfig)
		if err == nil && len(contexts) > 1 {
			chosen, quit := app.RunContextPicker(contexts, current)
			if quit {
				os.Exit(0)
			}
			cfg.Context = chosen
			if chosen != current {
				_ = k8s.PersistCurrentContext(cfg.Kubeconfig, chosen)
			}
		}
	}

	a, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kubecurses: init error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "kubecurses: %v\n", err)
		os.Exit(1)
	}
}
