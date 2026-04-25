// Package commands holds rules that only apply to slash-command
// artifacts.
package commands

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&allowedToolsKnown{}) }

// allowedToolsKnown errors when a slash command's allowed-tools list
// references a tool that is not in artifact.KnownTools. A typo in
// allowed-tools silently strips the tool from the command — worth an
// error-level signal.
type allowedToolsKnown struct{}

func (*allowedToolsKnown) ID() string                     { return "commands/allowed-tools-known" }
func (*allowedToolsKnown) Category() string               { return "schema" }
func (*allowedToolsKnown) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*allowedToolsKnown) DefaultOptions() map[string]any { return nil }
func (*allowedToolsKnown) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindCommand}
}

func (*allowedToolsKnown) HelpURI() string {
	return rules.DefaultHelpURI("commands/allowed-tools-known")
}

func (r *allowedToolsKnown) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	c, ok := a.(*artifact.Command)
	if !ok {
		return nil
	}
	if len(c.AllowedTools) == 0 {
		return nil
	}
	var out []diag.Diagnostic
	for _, tool := range c.AllowedTools {
		if artifact.IsKnownTool(tool) {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    c.Path(),
			Range:   c.Frontmatter.KeyRange("allowed-tools"),
			Message: fmt.Sprintf("unknown tool %q in allowed-tools", tool),
		})
	}
	return out
}
