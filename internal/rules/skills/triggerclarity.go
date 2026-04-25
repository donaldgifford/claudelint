package skills

import (
	"strings"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&triggerClarity{}) }

// triggerClarity warns when a skill's description does not contain
// an imperative trigger phrase. Without one, Claude has no natural
// signal for when to invoke the skill — "writes emails" is less
// actionable than "Use when drafting emails". The heuristic is a
// short list of verbs; the rule is advisory, not strict.
//
// The list is configurable via the `phrases` option so domain-
// specific skills can override it.
type triggerClarity struct{}

var defaultTriggerPhrases = []string{
	"use when",
	"trigger when",
	"invoke when",
	"when you",
	"if the user",
}

func (*triggerClarity) ID() string                     { return "skills/trigger-clarity" }
func (*triggerClarity) Category() string               { return "content" }
func (*triggerClarity) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*triggerClarity) DefaultOptions() map[string]any {
	return map[string]any{"phrases": defaultTriggerPhrases}
}

func (*triggerClarity) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindSkill}
}

func (*triggerClarity) HelpURI() string { return rules.DefaultHelpURI("skills/trigger-clarity") }

func (r *triggerClarity) Check(ctx rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.Skill)
	if !ok {
		return nil
	}
	if s.Description == "" {
		// schema/frontmatter-required already flags this; avoid
		// piling a second warning on the same artifact.
		return nil
	}
	phrases := stringSliceOption(ctx, "phrases", defaultTriggerPhrases)
	desc := strings.ToLower(s.Description)
	for _, p := range phrases {
		if strings.Contains(desc, strings.ToLower(p)) {
			return nil
		}
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    s.Path(),
		Range:   s.Frontmatter.KeyRange("description"),
		Message: "skill description has no trigger phrase (e.g. \"Use when ...\")",
	}}
}

// stringSliceOption coerces a rule option into a []string. Options
// coming from HCL land as []any of string; the default value is a
// []string.
func stringSliceOption(ctx rules.Context, key string, def []string) []string {
	v := ctx.Option(key)
	switch vv := v.(type) {
	case []string:
		return vv
	case []any:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return def
	}
}
