// Package cmd handles CLI flag parsing and wires up the application.
package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/1homsi/kubecurses/internal/app"
)

// Execute parses flags and runs the application. version is the build-time version string.
func Execute(version string) {
	cfg := app.DefaultConfig()

	showVersion := flag.Bool("version", false, "print version and exit")
	flag.StringVar(&cfg.Kubeconfig, "kubeconfig", "", "path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	flag.StringVar(&cfg.Context, "context", "", "kubernetes context to use")
	flag.StringVar(&cfg.Namespace, "namespace", "", "namespace to filter (default: all namespaces)")
	flag.DurationVar(&cfg.PollInterval, "interval", 10*time.Second, "polling interval for resource refresh")
	flag.Parse()

	if *showVersion {
		fmt.Println("kubecurses", version)
		os.Exit(0)
	}

	a, err := app.New(cfg)
	if err != nil {
		if err == app.ErrCancelled {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "kubecurses: init error: %v\n", err)
		os.Exit(1)
	}

	if err := a.Run(context.Background()); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "kubecurses: %v\n", err)
		os.Exit(1)
	}
}
