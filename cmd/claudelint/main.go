// Package main is the entry point for the claudelint CLI.
package main

import (
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
	code := cli.Execute(cli.BuildInfo{Version: version, Commit: commit}, os.Stderr)
	os.Exit(code)
}
