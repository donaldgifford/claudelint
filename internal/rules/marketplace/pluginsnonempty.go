package marketplace

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&pluginsNonempty{}) }

// pluginsNonempty warns when a marketplace declares no plugins. The
// manifest parses cleanly, but consumers would get an empty catalog;
// usually an authoring mistake.
type pluginsNonempty struct{}

func (*pluginsNonempty) ID() string                     { return "marketplace/plugins-nonempty" }
func (*pluginsNonempty) Category() string               { return categorySchema }
func (*pluginsNonempty) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*pluginsNonempty) DefaultOptions() map[string]any { return nil }
func (*pluginsNonempty) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (r *pluginsNonempty) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	if len(m.Plugins) > 0 {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    m.Path(),
		Message: "marketplace has an empty plugins array — catalog will be empty",
	}}
}
