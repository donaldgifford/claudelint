package cli

import (
	"github.com/spf13/cobra"
)

// newRulesCmd returns the `rules` subcommand. `rules` without args lists
// the registry; `rules <id>` prints rationale and default options. The
// registry itself arrives in phase 1.4.
func newRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rules [id]",
		Short: "List built-in rules or describe one by id",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			// TODO(phase-1.4): read from internal/rules registry.
			_, err := cmd.OutOrStdout().Write([]byte("0 rules registered\n"))
			return err
		},
	}
}
