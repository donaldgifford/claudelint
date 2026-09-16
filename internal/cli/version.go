package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/donaldgifford/claudelint/internal/rules"
	"github.com/donaldgifford/claudelint/internal/upstream/spec"
)

// newVersionCmd returns the `version` subcommand. Output shape matches
// DESIGN-0001, with the spec line added by DESIGN-0006:
//
//	claudelint <version> (<commit>)
//	ruleset    <ruleset-version> (<fingerprint>)
//	spec       <docs-marker> (<digest-fingerprint>)
//
// The three-line layout keeps each version pair visually aligned so
// operators can eyeball them at a glance; release notes reference any
// line independently.
//
// The spec line is here because it is the question a bug report cannot
// answer otherwise. "claudelint says my tool name is unknown" depends
// entirely on which documentation revision the binary was built
// against, and the ruleset version does not carry that.
func newVersionCmd(info BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print binary and ruleset versions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			specVersion, err := spec.Current()
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"claudelint %s (%s)\nruleset    %s (%s)\nspec       %s\n",
				info.Version, info.Commit,
				rules.RulesetVersion, rules.RulesetFingerprint(),
				specVersion,
			)

			return err
		},
	}
}
