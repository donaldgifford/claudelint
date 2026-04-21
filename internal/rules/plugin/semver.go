package plugin

import (
	"regexp"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&semverRule{}) }

// semverPattern is a pragmatic semver 2.0.0 check: MAJOR.MINOR.PATCH
// plus optional -pre+build parts. The regex rejects purely-numeric
// versions like "1" or "1.2" that plugin managers also treat as
// invalid in practice.
var semverPattern = regexp.MustCompile(
	`^v?\d+\.\d+\.\d+(-[A-Za-z0-9.-]+)?(\+[A-Za-z0-9.-]+)?$`,
)

// semverRule warns when a plugin manifest's version is not a valid
// semver string. Valid semver is not strictly required by Claude
// Code at install time but it is expected by marketplaces and
// auto-update tooling; warning-severity is enough.
type semverRule struct{}

func (*semverRule) ID() string                     { return "plugin/semver" }
func (*semverRule) Category() string               { return "schema" }
func (*semverRule) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*semverRule) DefaultOptions() map[string]any { return nil }
func (*semverRule) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindPlugin}
}

func (r *semverRule) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	p, ok := a.(*artifact.Plugin)
	if !ok {
		return nil
	}
	if p.Version == "" {
		// plugin/manifest-fields already flags this.
		return nil
	}
	if semverPattern.MatchString(p.Version) {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    p.Path(),
		Range:   p.VersionRange,
		Message: "plugin version is not valid semver (want MAJOR.MINOR.PATCH)",
	}}
}
