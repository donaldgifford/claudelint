package marketplace

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&pluginSourceValid{}) }

// pluginSourceValid errors when a plugins[] entry has no `source`
// field or an empty string. The "does the path exist on disk" check
// is deliberately out of scope here — rules are pure over the parsed
// artifact; filesystem validation belongs in a future engine-level
// pre-pass. A future sub-rule can extend this.
type pluginSourceValid struct{}

func (*pluginSourceValid) ID() string                     { return "marketplace/plugin-source-valid" }
func (*pluginSourceValid) Category() string               { return categorySchema }
func (*pluginSourceValid) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*pluginSourceValid) DefaultOptions() map[string]any { return nil }
func (*pluginSourceValid) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (r *pluginSourceValid) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	for i := range m.Plugins {
		p := &m.Plugins[i]
		if p.Source != "" {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    m.Path(),
			Range:   p.NameRange,
			Message: fmt.Sprintf("plugins[%d] is missing a non-empty source field", i),
		})
	}
	return out
}
