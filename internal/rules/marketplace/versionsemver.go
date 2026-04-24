package marketplace

import (
	"regexp"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&versionSemver{}) }

// semverPattern mirrors the plugin-manifest check: MAJOR.MINOR.PATCH
// with optional prerelease and build metadata. A leading "v" is
// accepted for parity with git-tag style.
var semverPattern = regexp.MustCompile(
	`^v?\d+\.\d+\.\d+(-[A-Za-z0-9.-]+)?(\+[A-Za-z0-9.-]+)?$`,
)

// versionSemver errors when the marketplace `version` field is
// missing or not a valid semver — marketplaces without a parseable
// version cannot be ordered by consumers.
type versionSemver struct{}

func (*versionSemver) ID() string                     { return "marketplace/version-semver" }
func (*versionSemver) Category() string               { return categorySchema }
func (*versionSemver) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*versionSemver) DefaultOptions() map[string]any { return nil }
func (*versionSemver) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (r *versionSemver) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	if m.Version == "" {
		return []diag.Diagnostic{{
			RuleID:  r.ID(),
			Path:    m.Path(),
			Message: `marketplace manifest is missing required field "version"`,
		}}
	}
	if semverPattern.MatchString(m.Version) {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    m.Path(),
		Range:   m.VersionRange,
		Message: "marketplace version is not valid semver (want MAJOR.MINOR.PATCH)",
	}}
}
