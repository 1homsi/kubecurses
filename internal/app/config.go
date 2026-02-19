// Package app wires together the UI and Kubernetes layers.
package app

import "time"

// Config holds runtime configuration derived from CLI flags.
type Config struct {
	Kubeconfig  string
	Context     string
	Namespace   string
	PollInterval time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		PollInterval: 10 * time.Second,
	}
}
