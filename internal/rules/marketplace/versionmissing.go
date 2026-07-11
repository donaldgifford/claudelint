package marketplace

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&versionMissing{}) }

// versionMissing nudges (info) when a marketplace manifest has no
// root `version`. The docs make the field optional, so this is not an
// error — but a declared version lets consumers order releases, so
// its absence is worth surfacing. The semver-shape check on declared
// versions is marketplace/version-semver.
type versionMissing struct{}

func (*versionMissing) ID() string                     { return "marketplace/version-missing" }
func (*versionMissing) Category() string               { return categoryStyle }
func (*versionMissing) DefaultSeverity() diag.Severity { return diag.SeverityInfo }
func (*versionMissing) DefaultOptions() map[string]any { return nil }
func (*versionMissing) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (*versionMissing) HelpURI() string { return rules.DefaultHelpURI("marketplace/version-missing") }

func (r *versionMissing) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok || m.Version != "" {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    m.Path(),
		Message: `marketplace manifest has no root "version"; declaring one lets consumers order releases`,
	}}
}
