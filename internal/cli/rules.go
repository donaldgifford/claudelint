package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// rulesJSONSchemaVersion is the schema version of the `claudelint
// rules --json` output. Bump when the shape changes in an
// incompatible way. See docs/rules-json-schema.md.
const rulesJSONSchemaVersion = "1"

// newRulesCmd returns the `rules` subcommand. Bare `claudelint rules`
// prints the registry as a table; `claudelint rules <id>` prints a
// detail view including AppliesTo and default options. `--json`
// switches the list output to a machine-readable schema documented
// in docs/rules-json-schema.md.
func newRulesCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "rules [id]",
		Short: "List built-in rules or describe one by id",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if len(args) == 0 {
				if asJSON {
					return listRulesJSON(out)
				}
				return listRules(out)
			}
			if asJSON {
				return describeRuleJSON(out, args[0])
			}
			return describeRule(out, args[0])
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON instead of a text table")
	return cmd
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
		fmt.Sprintf("  help:     %s", r.HelpURI()),
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

// ruleDoc is the per-rule payload emitted by `rules --json`. Field
// order, names, and types are the stable contract documented in
// docs/rules-json-schema.md — do not rename without bumping
// rulesJSONSchemaVersion.
type ruleDoc struct {
	ID              string         `json:"id"`
	Category        string         `json:"category"`
	DefaultSeverity string         `json:"default_severity"`
	AppliesTo       []string       `json:"applies_to"`
	HelpURI         string         `json:"help_uri"`
	DefaultOptions  map[string]any `json:"default_options"`
}

// rulesDoc is the envelope for `rules --json`. Same stability rules as
// ruleDoc.
type rulesDoc struct {
	SchemaVersion  string    `json:"schema_version"`
	RulesetVersion string    `json:"ruleset_version"`
	Fingerprint    string    `json:"fingerprint"`
	Rules          []ruleDoc `json:"rules"`
}

func listRulesJSON(out io.Writer) error {
	all := rules.All()
	docs := make([]ruleDoc, 0, len(all))
	for _, r := range all {
		docs = append(docs, toRuleDoc(r))
	}
	return writeRulesJSON(out, rulesDoc{
		SchemaVersion:  rulesJSONSchemaVersion,
		RulesetVersion: rules.RulesetVersion,
		Fingerprint:    rules.RulesetFingerprint(),
		Rules:          docs,
	})
}

func describeRuleJSON(out io.Writer, id string) error {
	r := rules.Get(id)
	if r == nil {
		return fmt.Errorf("unknown rule %q", id)
	}
	return writeRulesJSON(out, rulesDoc{
		SchemaVersion:  rulesJSONSchemaVersion,
		RulesetVersion: rules.RulesetVersion,
		Fingerprint:    rules.RulesetFingerprint(),
		Rules:          []ruleDoc{toRuleDoc(r)},
	})
}

func toRuleDoc(r rules.Rule) ruleDoc {
	kinds := r.AppliesTo()
	applies := make([]string, 0, len(kinds))
	for _, k := range kinds {
		applies = append(applies, string(k))
	}
	opts := r.DefaultOptions()
	if opts == nil {
		opts = map[string]any{}
	}
	return ruleDoc{
		ID:              r.ID(),
		Category:        r.Category(),
		DefaultSeverity: r.DefaultSeverity().String(),
		AppliesTo:       applies,
		HelpURI:         r.HelpURI(),
		DefaultOptions:  opts,
	}
}

func writeRulesJSON(out io.Writer, doc rulesDoc) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
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
