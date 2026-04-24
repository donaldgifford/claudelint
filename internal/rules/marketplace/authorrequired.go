package marketplace

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&authorRequired{}) }

// authorRequired emits an info-level diagnostic when the marketplace
// `author` field is missing. Not fatal — many public marketplaces
// omit it — but populated makes catalog listings more useful.
type authorRequired struct{}

func (*authorRequired) ID() string                     { return "marketplace/author-required" }
func (*authorRequired) Category() string               { return categoryStyle }
func (*authorRequired) DefaultSeverity() diag.Severity { return diag.SeverityInfo }
func (*authorRequired) DefaultOptions() map[string]any { return nil }
func (*authorRequired) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (*authorRequired) HelpURI() string { return rules.DefaultHelpURI("marketplace/author-required") }

func (r *authorRequired) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	if m.Author != "" {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    m.Path(),
		Message: `marketplace manifest has no "author" field`,
	}}
}
