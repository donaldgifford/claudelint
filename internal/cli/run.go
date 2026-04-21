package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/donaldgifford/claudelint/internal/discovery"
	"github.com/donaldgifford/claudelint/internal/reporter"
)

// newRunCmd returns the `run` subcommand, which lints the target
// paths. v1 wiring walks each path, discovers artifacts, and emits a
// text-format summary. Parsing and rule execution land in phases 1.2
// and 1.4+; for phase 1.1 the diagnostic list is always empty so users
// get a working "0 diagnostics, N files checked" end-to-end today.
func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [path...]",
		Short: "Lint Claude artifacts under the given paths (default: cwd)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			targets := args
			if len(targets) == 0 {
				targets = []string{"."}
			}

			w := discovery.New(discovery.Options{})
			var files int
			for _, t := range targets {
				cands, err := w.Walk(t)
				if err != nil {
					return fmt.Errorf("discover %s: %w", t, err)
				}
				files += len(cands)
			}
			return reporter.Text(cmd.OutOrStdout(), reporter.Summary{Files: files})
		},
	}
}
