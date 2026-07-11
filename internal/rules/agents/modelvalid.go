// Package agents holds rules for subagent artifacts. Where the docs
// share a field across the merged frontmatter models — `model` also
// exists on skills and commands — a rule here runs on those kinds
// too (see modelValid).
package agents

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// Shared category constants, declared once per package so rules can
// reference them without triggering goconst.
const (
	categorySchema  = "schema"
	categoryContent = "content"
)

func init() { rules.Register(&modelValid{}) }

// modelValid warns when a declared `model` value is not a documented
// reference: an alias (sonnet/opus/haiku/fable), "inherit", or a full
// model ID (claude-...). A typo'd value silently falls back to the
// inherited model, so the mistake never surfaces at runtime. Absent
// keys are fine — omitted means inherit.
type modelValid struct{}

func (*modelValid) ID() string                     { return "agents/model-valid" }
func (*modelValid) Category() string               { return categorySchema }
func (*modelValid) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*modelValid) DefaultOptions() map[string]any { return nil }
func (*modelValid) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindAgent, artifact.KindSkill, artifact.KindCommand}
}

func (*modelValid) HelpURI() string { return rules.DefaultHelpURI("agents/model-valid") }

func (r *modelValid) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	var model string
	var fm *artifact.Frontmatter
	switch v := a.(type) {
	case *artifact.Agent:
		model, fm = v.Model, &v.Frontmatter
	case *artifact.Skill:
		model, fm = v.Model, &v.Frontmatter
	case *artifact.Command:
		model, fm = v.Model, &v.Frontmatter
	default:
		return nil
	}
	if model == "" || artifact.IsValidModelRef(model) {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID: r.ID(),
		Path:   a.Path(),
		Range:  fm.KeyRange("model"),
		Message: fmt.Sprintf(
			"model %q is not a known value; want sonnet, opus, haiku, fable, inherit, or a full model ID (claude-...)",
			model),
	}}
}
