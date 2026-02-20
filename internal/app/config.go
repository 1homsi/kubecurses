// Package app wires together the UI and Kubernetes layers.
package app

import "time"

// Config holds runtime configuration derived from CLI flags.
type Config struct {
	Kubeconfig      string
	Context         string
	Namespace       string
	Watch           bool          // default true — use informers instead of polling
	PollInterval    time.Duration // default 10s (polling mode only)
	MetricsInterval time.Duration // default 30s
	RequestTimeout  time.Duration // default 30s
	KubeAPIQPS      float32       // default 20
	KubeAPIBurst    int           // default 40
	DisableMetrics  bool
	MaxPods         int   // 0 = unlimited
	NoIcons         bool
	LogTailLines    int64 // default 200
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Watch:           true,
		PollInterval:    10 * time.Second,
		MetricsInterval: 30 * time.Second,
		RequestTimeout:  30 * time.Second,
		KubeAPIQPS:      20,
		KubeAPIBurst:    40,
		LogTailLines:    200,
	}
}
