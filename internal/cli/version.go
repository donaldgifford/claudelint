package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/donaldgifford/claudelint/internal/rules"
)

// newVersionCmd returns the `version` subcommand. Output shape matches
// DESIGN-0001:
//
//	claudelint <version> (<commit>)
//	ruleset    <ruleset-version> (<fingerprint>)
//
// The two-line layout keeps each version pair visually aligned so
// operators can eyeball both at a glance; release notes reference
// either line independently.
func newVersionCmd(info BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print binary and ruleset versions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"claudelint %s (%s)\nruleset    %s (%s)\n",
				info.Version, info.Commit,
				rules.RulesetVersion, rules.RulesetFingerprint(),
			)
			return err
		},
	}
}
