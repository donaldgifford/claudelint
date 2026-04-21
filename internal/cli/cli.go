// Package cli wires the claudelint cobra command tree.
//
// The CLI entrypoint in cmd/claudelint is deliberately thin — it only
// translates ldflags into a BuildInfo and hands control to Execute. All
// subcommand wiring and flag parsing lives here so it can be tested
// without spawning a process.
package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// BuildInfo carries version metadata injected by the build. It is
// populated from -ldflags in cmd/claudelint/main.go.
type BuildInfo struct {
	Version string
	Commit  string
}

// Execute runs the root command with the given build info and returns
// a process exit code. It is the entry point main uses: exit 0 on
// clean runs, exit 1 when diagnostics failed the run, exit 2 on usage
// / config / I/O errors. stderr receives the human-readable error
// text for the exit 2 path.
func Execute(info BuildInfo, stderr io.Writer) int {
	err := newRootCmd(info).Execute()
	if err == nil {
		return exitSuccess
	}
	if errors.Is(err, errDiagnostics) {
		return exitHasErrors
	}
	if _, werr := fmt.Fprintln(stderr, err); werr != nil {
		// Stderr write failure is terminal; nothing useful to do.
		return exitUsage
	}
	return exitUsage
}

// newRootCmd assembles the claudelint command tree. Bare `claudelint`
// delegates to `run` so `claudelint .` works the way users expect.
func newRootCmd(info BuildInfo) *cobra.Command {
	run := newRunCmd()

	root := &cobra.Command{
		Use:   "claudelint [path...]",
		Short: "Lint Claude Code artifacts (CLAUDE.md, skills, commands, agents, hooks, plugins)",
		Long: "claudelint is a static linter for Claude Code artifacts. It discovers " +
			"CLAUDE.md files, skills, slash commands, subagents, hooks, and plugin " +
			"manifests in a repository, parses each into a typed value, and runs a " +
			"built-in ruleset against them.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          run.RunE,
		Args:          run.Args,
	}

	root.AddCommand(run)
	root.AddCommand(newRulesCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newVersionCmd(info))

	return root
}
