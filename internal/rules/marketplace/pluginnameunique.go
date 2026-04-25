package marketplace

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&pluginNameUnique{}) }

// pluginNameUnique errors when two plugins[] entries share a name.
// Duplicate names silently shadow each other at install time — one of
// them will never surface in the consumer's UI.
type pluginNameUnique struct{}

func (*pluginNameUnique) ID() string                     { return "marketplace/plugin-name-unique" }
func (*pluginNameUnique) Category() string               { return categorySchema }
func (*pluginNameUnique) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*pluginNameUnique) DefaultOptions() map[string]any { return nil }
func (*pluginNameUnique) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (*pluginNameUnique) HelpURI() string {
	return rules.DefaultHelpURI("marketplace/plugin-name-unique")
}

func (r *pluginNameUnique) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	seen := make(map[string]int, len(m.Plugins))
	var out []diag.Diagnostic
	for i := range m.Plugins {
		p := &m.Plugins[i]
		if p.Name == "" {
			continue
		}
		if prev, dup := seen[p.Name]; dup {
			out = append(out, diag.Diagnostic{
				RuleID:  r.ID(),
				Path:    m.Path(),
				Range:   p.NameRange,
				Message: fmt.Sprintf("plugins[%d].name %q duplicates plugins[%d].name", i, p.Name, prev),
			})
			continue
		}
		seen[p.Name] = i
	}
	return out
}
