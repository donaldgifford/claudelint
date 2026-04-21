package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// newRulesCmd returns the `rules` subcommand. Bare `claudelint rules`
// prints the registry as a table; `claudelint rules <id>` prints a
// detail view including AppliesTo and default options.
func newRulesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rules [id]",
		Short: "List built-in rules or describe one by id",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if len(args) == 0 {
				return listRules(out)
			}
			return describeRule(out, args[0])
		},
	}
}

func listRules(out io.Writer) error {
	all := rules.All()
	if len(all) == 0 {
		_, err := fmt.Fprintln(out, "0 rules registered")
		return err
	}
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ID\tCATEGORY\tSEVERITY\tKINDS"); err != nil {
		return err
	}
	for _, r := range all {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			r.ID(), r.Category(), r.DefaultSeverity(), kindsOf(r)); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func describeRule(out io.Writer, id string) error {
	r := rules.Get(id)
	if r == nil {
		return fmt.Errorf("unknown rule %q", id)
	}
	lines := []string{
		r.ID(),
		fmt.Sprintf("  category: %s", r.Category()),
		fmt.Sprintf("  severity: %s", r.DefaultSeverity()),
		fmt.Sprintf("  applies:  %s", kindsOf(r)),
	}
	opts := r.DefaultOptions()
	if len(opts) == 0 {
		lines = append(lines, "  options:  (none)")
	} else {
		lines = append(lines, "  options:")
		keys := make([]string, 0, len(opts))
		for k := range opts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("    %s = %v", k, opts[k]))
		}
	}
	_, err := fmt.Fprintln(out, strings.Join(lines, "\n"))
	return err
}

// kindsOf returns a comma-joined list of the rule's artifact kinds.
// "*" is shorthand for "applies to every kind".
func kindsOf(r rules.Rule) string {
	kinds := r.AppliesTo()
	if len(kinds) == 0 || len(kinds) == len(artifact.AllKinds()) {
		return "*"
	}
	parts := make([]string, 0, len(kinds))
	for _, k := range kinds {
		parts = append(parts, string(k))
	}
	return strings.Join(parts, ",")
}
