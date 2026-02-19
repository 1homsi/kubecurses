package main

import "github.com/1homsi/kubecurses/cmd"

// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	cmd.Execute(version)
}
