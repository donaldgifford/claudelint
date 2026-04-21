package cli

import (
	"github.com/spf13/cobra"
)

// newRunCmd returns the `run` subcommand, which lints the target paths.
// v1 wiring is a stub; phases 1.2–1.7 will fill it in.
func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [path...]",
		Short: "Lint Claude artifacts under the given paths (default: cwd)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// TODO(phase-1.1): wire discovery → reporter; print counts.
			_, err := cmd.OutOrStdout().Write([]byte("0 diagnostics, 0 files checked\n"))
			return err
		},
	}
}
