package skills

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&noVersionField{}) }

// noVersionField warns when a SKILL.md frontmatter declares a
// `version` key. Skill versioning is load-bearing only at the plugin
// level (plugin.json's `version` field); a `version:` in SKILL.md
// frontmatter is non-standard, silently ignored by Claude Code, and
// creates two competing sources of truth that drift over time.
//
// Default severity is warning so the rule does not block CI by
// default. Teams that want to enforce the convention can upgrade to
// error in `.claudelint.hcl`. The rule has no options.
type noVersionField struct{}

func (*noVersionField) ID() string                     { return "skills/no-version-field" }
func (*noVersionField) Category() string               { return "schema" }
func (*noVersionField) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*noVersionField) DefaultOptions() map[string]any { return nil }
func (*noVersionField) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindSkill}
}

func (*noVersionField) HelpURI() string { return rules.DefaultHelpURI("skills/no-version-field") }

func (r *noVersionField) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.Skill)
	if !ok {
		return nil
	}
	rng := s.Frontmatter.KeyRange("version")
	if rng.IsZero() {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    s.Path(),
		Range:   rng,
		Message: "SKILL.md frontmatter has 'version' field; version belongs in plugin.json",
	}}
}
