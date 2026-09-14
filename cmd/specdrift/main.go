// Package main is the entry point for specdrift, the development tool
// that detects drift between claudelint's ruleset and the upstream
// Claude Code and Agent Skills specifications.
//
// It is not shipped: goreleaser builds only ./cmd/claudelint. Run it
// with "go run ./cmd/specdrift", or through "just spec-check" and
// "just spec-sync".
package main

import (
	"os"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

func main() {
	os.Exit(upstream.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
