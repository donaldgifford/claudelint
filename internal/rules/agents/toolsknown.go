package agents

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&toolsKnown{}) }

// toolsKnown warns when an entry in an agent's tools or
// disallowedTools list is neither a known tool name nor a valid
// pattern (mcp__* references, permission-rule forms). Claude Code
// silently ignores unknown names, so a typo widens or narrows the
// agent's tool access without any runtime signal.
type toolsKnown struct{}

func (*toolsKnown) ID() string                     { return "agents/tools-known" }
func (*toolsKnown) Category() string               { return categorySchema }
func (*toolsKnown) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*toolsKnown) DefaultOptions() map[string]any { return nil }
func (*toolsKnown) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindAgent}
}

func (*toolsKnown) HelpURI() string { return rules.DefaultHelpURI("agents/tools-known") }

func (r *toolsKnown) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	ag, ok := a.(*artifact.Agent)
	if !ok {
		return nil
	}
	return append(
		r.checkList(ag, "tools", ag.Tools),
		r.checkList(ag, "disallowedTools", ag.DisallowedTools)...)
}

// checkList flags entries in one tool-list key that fail both the
// known-name and pattern classifiers.
func (r *toolsKnown) checkList(ag *artifact.Agent, key string, tools []string) []diag.Diagnostic {
	var out []diag.Diagnostic
	for _, tool := range tools {
		if artifact.IsKnownTool(tool) || artifact.IsToolPattern(tool) {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID: r.ID(),
			Path:   ag.Path(),
			Range:  ag.Frontmatter.KeyRange(key),
			Message: fmt.Sprintf(
				"unknown tool %q in %s — Claude Code silently ignores unknown names", tool, key),
		})
	}
	return out
}
