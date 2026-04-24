package marketplace

import (
	"fmt"
	"path"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&pluginNameMatchesDir{}) }

// pluginNameMatchesDir warns when a plugin's declared name does not
// match the basename of its source directory. A pure UX guardrail —
// works without divergence, but mismatched names confuse users who
// look at the filesystem and the catalog side by side.
//
// Skipped for external sources (Resolved is empty) since their
// directory basename is not locally meaningful.
type pluginNameMatchesDir struct{}

func (*pluginNameMatchesDir) ID() string                     { return "marketplace/plugin-name-matches-dir" }
func (*pluginNameMatchesDir) Category() string               { return categoryStyle }
func (*pluginNameMatchesDir) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*pluginNameMatchesDir) DefaultOptions() map[string]any { return nil }
func (*pluginNameMatchesDir) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (r *pluginNameMatchesDir) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	for i := range m.Plugins {
		p := &m.Plugins[i]
		if p.Name == "" || p.Resolved == "" {
			continue
		}
		basename := path.Base(p.Resolved)
		// For source: "./" the basename is ".", which carries no
		// meaningful comparison — skip rather than noise-warn.
		if basename == "." || basename == p.Name {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    m.Path(),
			Range:   p.NameRange,
			Message: fmt.Sprintf("plugins[%d].name %q does not match source directory basename %q", i, p.Name, basename),
		})
	}
	return out
}
