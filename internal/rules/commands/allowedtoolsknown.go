// Package commands holds rules for slash-command artifacts. Where the
// docs define one frontmatter model for commands and skills, a rule
// here may also run on skills (see allowedToolsKnown).
package commands

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&allowedToolsKnown{}) }

// allowedToolsKnown errors when an allowed-tools or disallowed-tools
// list references a tool that is neither in artifact.KnownTools nor a
// structurally valid pattern (permission rules like Bash(git add:*),
// mcp__* references). A typo silently strips the tool — worth an
// error-level signal. Commands and skills share one frontmatter
// model, so the rule runs on both kinds; the ID keeps its historical
// commands/ prefix.
type allowedToolsKnown struct{}

func (*allowedToolsKnown) ID() string                     { return "commands/allowed-tools-known" }
func (*allowedToolsKnown) Category() string               { return "schema" }
func (*allowedToolsKnown) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*allowedToolsKnown) DefaultOptions() map[string]any { return nil }
func (*allowedToolsKnown) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindCommand, artifact.KindSkill}
}

func (*allowedToolsKnown) HelpURI() string {
	return rules.DefaultHelpURI("commands/allowed-tools-known")
}

func (r *allowedToolsKnown) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	switch v := a.(type) {
	case *artifact.Command:
		return append(
			r.checkList(v.Path(), &v.Frontmatter, "allowed-tools", v.AllowedTools),
			r.checkList(v.Path(), &v.Frontmatter, "disallowed-tools", v.DisallowedTools)...)
	case *artifact.Skill:
		return append(
			r.checkList(v.Path(), &v.Frontmatter, "allowed-tools", v.AllowedTools),
			r.checkList(v.Path(), &v.Frontmatter, "disallowed-tools", v.DisallowedTools)...)
	default:
		return nil
	}
}

// checkList flags entries in one tool-list key that are neither known
// tool names nor valid patterns per artifact.IsToolPattern.
func (r *allowedToolsKnown) checkList(
	path string, fm *artifact.Frontmatter, key string, tools []string,
) []diag.Diagnostic {
	var out []diag.Diagnostic
	for _, tool := range tools {
		if artifact.IsKnownTool(tool) || artifact.IsToolPattern(tool) {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    path,
			Range:   fm.KeyRange(key),
			Message: fmt.Sprintf("unknown tool %q in %s", tool, key),
		})
	}
	return out
}
