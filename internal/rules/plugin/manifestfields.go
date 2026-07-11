// Package plugin holds rules that apply to plugin manifest
// artifacts.
package plugin

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&manifestFields{}) }

// manifestFields errors when a plugin manifest is missing `name` or
// `version`. The docs require only `name` (a missing `version` falls
// back to the installing commit's git SHA); requiring `version` too
// is claudelint's stricter-by-design stance — pinned versions make
// update semantics explicit for marketplace review (OQ3; see
// rules.md).
type manifestFields struct{}

func (*manifestFields) ID() string                     { return "plugin/manifest-fields" }
func (*manifestFields) Category() string               { return "schema" }
func (*manifestFields) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*manifestFields) DefaultOptions() map[string]any { return nil }
func (*manifestFields) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindPlugin}
}

func (*manifestFields) HelpURI() string { return rules.DefaultHelpURI("plugin/manifest-fields") }

func (r *manifestFields) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	p, ok := a.(*artifact.Plugin)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	if p.Name == "" {
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    p.Path(),
			Message: `plugin manifest is missing required field "name"`,
		})
	}
	if p.Version == "" {
		out = append(out, diag.Diagnostic{
			RuleID: r.ID(),
			Path:   p.Path(),
			Message: `plugin manifest has no "version" — claudelint requires a pinned version ` +
				`(Claude Code would fall back to the git SHA)`,
		})
	}
	return out
}
