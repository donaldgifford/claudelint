package marketplace

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&ownerRequired{}) }

// ownerRequired warns when a marketplace manifest identifies no
// maintainer at all: the docs require root owner{name}, and the
// legacy top-level author string (still documented backward-compat)
// satisfies the requirement too. When only the legacy form is
// present, the separate marketplace/author-legacy rule nudges toward
// the owner{} shape. This replaces the pre-v1.3.0
// marketplace/author-required rule.
type ownerRequired struct{}

func (*ownerRequired) ID() string                     { return "marketplace/owner-required" }
func (*ownerRequired) Category() string               { return categorySchema }
func (*ownerRequired) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*ownerRequired) DefaultOptions() map[string]any { return nil }
func (*ownerRequired) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (*ownerRequired) HelpURI() string { return rules.DefaultHelpURI("marketplace/owner-required") }

func (r *ownerRequired) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	// Author is the merged view: it carries owner.name when only the
	// object form is declared, so either field being set means the
	// manifest names a maintainer.
	if m.OwnerName != "" || m.Author != "" {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    m.Path(),
		Message: `marketplace manifest is missing "owner" — the docs require root owner{name}`,
	}}
}
