// Package cli wires the claudelint cobra command tree.
//
// The CLI entrypoint in cmd/claudelint is deliberately thin — it only
// translates ldflags into a BuildInfo and hands control to Execute. All
// subcommand wiring and flag parsing lives here so it can be tested
// without spawning a process.
package cli

import (
	"github.com/spf13/cobra"
)

// BuildInfo carries version metadata injected by the build. It is
// populated from -ldflags in cmd/claudelint/main.go.
type BuildInfo struct {
	Version string
	Commit  string
}

// Execute runs the root command with the given build info. It returns
// any error cobra surfaces so main can print it and set a non-zero exit
// code.
func Execute(info BuildInfo) error {
	return newRootCmd(info).Execute()
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
