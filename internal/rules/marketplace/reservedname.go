package marketplace

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&reservedName{}) }

// reservedName errors when a marketplace uses one of the documented
// reserved names.
type reservedName struct{}

func (*reservedName) ID() string                     { return "marketplace/reserved-name" }
func (*reservedName) Category() string               { return categorySchema }
func (*reservedName) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*reservedName) DefaultOptions() map[string]any { return nil }
func (*reservedName) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (*reservedName) HelpURI() string { return rules.DefaultHelpURI("marketplace/reserved-name") }

func (r *reservedName) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	if _, reserved := artifact.ReservedMarketplaceNames[m.Name]; !reserved {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID: r.ID(),
		Path:   m.Path(),
		Range:  m.NameRange,
		Message: fmt.Sprintf(
			"marketplace name %q is reserved for official Anthropic use — Claude Code refuses to load it",
			m.Name),
	}}
}
