// Package k8s provides Kubernetes API access and background watchers.
package k8s

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// ClientOptions configures the Kubernetes REST client.
type ClientOptions struct {
	KubeconfigPath string
	ContextName    string
	RequestTimeout time.Duration // 0 = no timeout
	QPS            float32       // 0 = use client-go default
	Burst          int           // 0 = use client-go default
}

// NewClient builds a *kubernetes.Clientset from the given options.
// If KubeconfigPath is empty, the default discovery order
// (KUBECONFIG env var → ~/.kube/config) is used.
func NewClient(opts ClientOptions) (*kubernetes.Clientset, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.KubeconfigPath != "" {
		loadingRules.ExplicitPath = opts.KubeconfigPath
	}

	overrides := &clientcmd.ConfigOverrides{}
	if opts.ContextName != "" {
		overrides.CurrentContext = opts.ContextName
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		overrides,
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig: %w", err)
	}

	if opts.RequestTimeout > 0 {
		config.Timeout = opts.RequestTimeout
	}
	if opts.QPS > 0 {
		config.QPS = opts.QPS
	}
	if opts.Burst > 0 {
		config.Burst = opts.Burst
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}
	return cs, nil
}
