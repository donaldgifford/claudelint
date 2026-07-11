package marketplace

import (
	"fmt"
	"regexp"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&nameFormat{}) }

// kebabCasePattern is the documented shape for marketplace and plugin
// entry names: lowercase alphanumeric segments joined by single
// hyphens ("acme-tools", "helm2"). claude.ai sync rejects violations.
var kebabCasePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// nameFormat warns when the marketplace name or a plugins[] entry
// name is not kebab-case. Names are public-facing (`/plugin install
// <plugin>@<marketplace>`) and claude.ai sync rejects nonconforming
// ones. Empty names are left to marketplace/name and
// plugin-source-valid.
type nameFormat struct{}

func (*nameFormat) ID() string                     { return "marketplace/name-format" }
func (*nameFormat) Category() string               { return categoryStyle }
func (*nameFormat) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*nameFormat) DefaultOptions() map[string]any { return nil }
func (*nameFormat) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (*nameFormat) HelpURI() string { return rules.DefaultHelpURI("marketplace/name-format") }

func (r *nameFormat) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	if m.Name != "" && !kebabCasePattern.MatchString(m.Name) {
		out = append(out, diag.Diagnostic{
			RuleID: r.ID(),
			Path:   m.Path(),
			Range:  m.NameRange,
			Message: fmt.Sprintf(
				"marketplace name %q should be kebab-case (lowercase letters, digits, hyphens)",
				m.Name),
		})
	}
	for i := range m.Plugins {
		p := &m.Plugins[i]
		if p.Name == "" || kebabCasePattern.MatchString(p.Name) {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID: r.ID(),
			Path:   m.Path(),
			Range:  p.NameRange,
			Message: fmt.Sprintf(
				"plugin name %q should be kebab-case (lowercase letters, digits, hyphens)",
				p.Name),
		})
	}
	return out
}
