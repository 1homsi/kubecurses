package k8s

import (
	"sort"

	"k8s.io/client-go/tools/clientcmd"
)

// ListContexts returns all context names from the active kubeconfig, sorted
// alphabetically, and the name of the currently active context.
func ListContexts(kubeconfigPath string) (contexts []string, current string, err error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	cfg, err := loadingRules.Load()
	if err != nil {
		return nil, "", err
	}

	for name := range cfg.Contexts {
		contexts = append(contexts, name)
	}
	sort.Strings(contexts)

	return contexts, cfg.CurrentContext, nil
}
