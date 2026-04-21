package cli

import (
	"github.com/spf13/cobra"
)

// newInitCmd returns the `init` subcommand, which scaffolds a default
// .claudelint.hcl. The actual scaffold lives in phase 1.3's config
// package; this stub keeps the command surface stable.
func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Scaffold a default .claudelint.hcl in the current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// TODO(phase-1.3): write a commented default .claudelint.hcl.
			_, err := cmd.OutOrStdout().Write([]byte("init: not yet implemented\n"))
			return err
		},
	}
}
