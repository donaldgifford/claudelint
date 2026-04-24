package marketplace

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&externalSourceSkipped{}) }

// externalSourceSkipped emits an info diagnostic for each plugins[]
// entry whose source is external (git URL). The claudelint linter
// does not fetch or lint external sources in Phase 2 — this rule
// surfaces the entry so users know it was noticed and skipped, not
// silently ignored. Opt-in remote fetching may arrive in a later
// phase.
//
// An entry is external when it has a non-empty Source but an empty
// Resolved (the parser only blanks Resolved for github:, https://,
// git@, and similar patterns).
type externalSourceSkipped struct{}

func (*externalSourceSkipped) ID() string                     { return "marketplace/external-source-skipped" }
func (*externalSourceSkipped) Category() string               { return categorySchema }
func (*externalSourceSkipped) DefaultSeverity() diag.Severity { return diag.SeverityInfo }
func (*externalSourceSkipped) DefaultOptions() map[string]any { return nil }
func (*externalSourceSkipped) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (r *externalSourceSkipped) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	for i := range m.Plugins {
		p := &m.Plugins[i]
		if p.Source == "" || p.Resolved != "" {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    m.Path(),
			Range:   p.SourceRange,
			Message: fmt.Sprintf("plugins[%d].source %q is external — skipped (not fetched by claudelint)", i, p.Source),
		})
	}
	return out
}
