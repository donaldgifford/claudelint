package marketplace

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&reservedName{}) }

// reservedMarketplaceNames mirrors the 16 names the plugin-marketplaces
// reference reserves for official Anthropic use (as of Claude Code
// v2.1.205). Claude Code re-checks the list every time it loads a
// marketplace, so a manifest shipping one of these stops loading for
// every user. Exact match only — impersonation heuristics
// ("official-claude-plugins") are deliberately not attempted here;
// claude.ai enforces those server-side.
var reservedMarketplaceNames = map[string]struct{}{
	"claude-code-marketplace":       {},
	"claude-code-plugins":           {},
	"claude-plugins-official":       {},
	"claude-plugins-community":      {},
	"claude-community":              {},
	"anthropic-marketplace":         {},
	"anthropic-plugins":             {},
	"agent-skills":                  {},
	"anthropic-agent-skills":        {},
	"knowledge-work-plugins":        {},
	"life-sciences":                 {},
	"claude-for-legal":              {},
	"claude-for-financial-services": {},
	"financial-services-plugins":    {},
	"first-party-plugins":           {},
	"healthcare":                    {},
}

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
	if _, reserved := reservedMarketplaceNames[m.Name]; !reserved {
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
