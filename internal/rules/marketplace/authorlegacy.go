package marketplace

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&authorLegacy{}) }

// authorLegacy nudges (info) when a marketplace identifies its
// maintainer only through the legacy top-level author string. The
// documented shape is owner{name,email}; author still works, so this
// is a rename hint, not an error. Manifests that declare owner{} are
// silent here even if they also keep author around.
type authorLegacy struct{}

func (*authorLegacy) ID() string                     { return "marketplace/author-legacy" }
func (*authorLegacy) Category() string               { return categoryStyle }
func (*authorLegacy) DefaultSeverity() diag.Severity { return diag.SeverityInfo }
func (*authorLegacy) DefaultOptions() map[string]any { return nil }
func (*authorLegacy) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (*authorLegacy) HelpURI() string { return rules.DefaultHelpURI("marketplace/author-legacy") }

func (r *authorLegacy) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok || m.OwnerName != "" || m.Author == "" {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    m.Path(),
		Range:   m.AuthorRange,
		Message: `legacy "author" field; the documented shape is "owner": {"name": ..., "email": ...}`,
	}}
}
