package skills

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&descriptionLength{}) }

// categoryContent is shared by this package's content rules (goconst).
const categoryContent = "content"

// defaultMaxDescriptionChars is the documented combined budget for
// description + when_to_use: both are injected into the system prompt
// for skill selection, and Claude Code truncates past 1024 tokens —
// the docs put the practical character ceiling at 1536.
const defaultMaxDescriptionChars = 1536

// descriptionLength warns when a skill's `description` and
// `when_to_use` together exceed the documented character budget.
// Overlong trigger text gets truncated in the selection prompt, so
// the tail — usually the "when to use" detail — silently stops
// influencing skill selection.
//
// Options:
//
//	max_chars (int): combined character budget, default 1536.
type descriptionLength struct{}

func (*descriptionLength) ID() string                     { return "skills/description-length" }
func (*descriptionLength) Category() string               { return categoryContent }
func (*descriptionLength) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*descriptionLength) DefaultOptions() map[string]any {
	return map[string]any{"max_chars": defaultMaxDescriptionChars}
}

func (*descriptionLength) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindSkill}
}

func (*descriptionLength) HelpURI() string {
	return rules.DefaultHelpURI("skills/description-length")
}

func (r *descriptionLength) Check(ctx rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.Skill)
	if !ok {
		return nil
	}
	maxChars := intOption(ctx, "max_chars", defaultMaxDescriptionChars)
	total := len(s.Description) + len(s.WhenToUse)
	if total <= maxChars {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID: r.ID(),
		Path:   s.Path(),
		Range:  s.Frontmatter.KeyRange("description"),
		Message: fmt.Sprintf(
			"description + when_to_use is %d chars (limit %d) — the tail is truncated in the skill-selection prompt",
			total, maxChars),
	}}
}
