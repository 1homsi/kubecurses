package k8s

import (
	"errors"
	"fmt"
	"os/exec"
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
//
// It delegates to `kubectl config use-context` because clientcmd.ModifyConfig has
// ambiguous write behaviour when KUBECONFIG contains multiple files. kubectl's own
// implementation always picks the right file. If kubectl is not in PATH the function
// falls back to the clientcmd API.
func PersistCurrentContext(kubeconfigPath, contextName string) error {
	args := []string{"config", "use-context", contextName}
	if kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", kubeconfigPath}, args...)
	}
	err := exec.Command("kubectl", args...).Run()
	if err == nil {
		return nil
	}
	// kubectl not found — fall back to clientcmd API.
	if errors.Is(err, exec.ErrNotFound) {
		return persistCurrentContextAPI(kubeconfigPath, contextName)
	}
	return fmt.Errorf("kubectl config use-context %s: %w", contextName, err)
}

func persistCurrentContextAPI(kubeconfigPath, contextName string) error {
	pathOptions := clientcmd.NewDefaultPathOptions()
	if kubeconfigPath != "" {
		pathOptions.LoadingRules.ExplicitPath = kubeconfigPath
	}
	config, err := pathOptions.GetStartingConfig()
	if err != nil {
		return fmt.Errorf("read kubeconfig: %w", err)
	}
	config.CurrentContext = contextName
	if err := clientcmd.ModifyConfig(pathOptions, *config, true); err != nil {
		return fmt.Errorf("write kubeconfig: %w", err)
	}
	return nil
}
