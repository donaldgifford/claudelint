// Package marketplace holds rules that apply to
// .claude-plugin/marketplace.json manifests. Rules in this package
// operate on *artifact.Marketplace values produced by
// artifact.ParseMarketplace — they never touch the filesystem or
// re-parse the JSON.
package marketplace

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// Shared category names. Using constants here both documents the
// rule categorization and keeps goconst happy.
const (
	categorySchema = "schema"
	categoryStyle  = "style"
)

func init() { rules.Register(&name{}) }

// name errors when a marketplace manifest is missing its top-level
// `name` field. Catalogs cannot display a marketplace without one.
type name struct{}

func (*name) ID() string                     { return "marketplace/name" }
func (*name) Category() string               { return categorySchema }
func (*name) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*name) DefaultOptions() map[string]any { return nil }
func (*name) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (r *name) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	if m.Name != "" {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    m.Path(),
		Message: `marketplace manifest is missing required field "name"`,
	}}
}
