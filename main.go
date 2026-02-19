package main

import "github.com/1homsi/kubecurses/cmd"

// These variables are set at build time via -ldflags.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	cmd.Execute(version, commit, buildDate)
}
