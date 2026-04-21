// Package main is the entry point for the claudelint CLI.
package main

import (
	"fmt"
	"os"

	"github.com/donaldgifford/claudelint/internal/cli"
	// Blank-import every rule subpackage so each rule's init()
	// registers exactly once at binary startup.
	_ "github.com/donaldgifford/claudelint/internal/rules/all"
)

// version is injected via -ldflags at build time.
var (
	version = "dev"
	commit  = "none"
)

func main() {
	if err := cli.Execute(cli.BuildInfo{Version: version, Commit: commit}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
