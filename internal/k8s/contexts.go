package k8s

import (
	"fmt"
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

// PersistCurrentContext writes contextName as the current-context in the kubeconfig
// so that kubectl and other tools see the same context after kubecurses exits.
// It respects kubeconfigPath if non-empty, otherwise uses the default kubeconfig.
func PersistCurrentContext(kubeconfigPath, contextName string) error {
	pathOptions := clientcmd.NewDefaultPathOptions()
	if kubeconfigPath != "" {
		pathOptions.LoadingRules.ExplicitPath = kubeconfigPath
	}
	config, err := pathOptions.GetStartingConfig()
	if err != nil {
		return fmt.Errorf("read kubeconfig: %w", err)
	}
	config.CurrentContext = contextName
	if err := clientcmd.ModifyConfig(pathOptions, *config, false); err != nil {
		return fmt.Errorf("write kubeconfig: %w", err)
	}
	return nil
}
