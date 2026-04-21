package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// rulesetVersion and rulesetFingerprint are stand-ins until phase 1.4
// moves them into internal/rules. They are defined here so the `version`
// subcommand's output shape — `claudelint vX.Y.Z (commit) rules vA.B.C
// (fingerprint)` — is fixed from phase 1.1 onward.
const (
	rulesetVersion     = "v0.0.0"
	rulesetFingerprint = "unset"
)

// newVersionCmd returns the `version` subcommand.
func newVersionCmd(info BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print binary and ruleset versions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"claudelint %s (%s) rules %s (%s)\n",
				info.Version, info.Commit,
				rulesetVersion, rulesetFingerprint,
			)
			return err
		},
	}
}
