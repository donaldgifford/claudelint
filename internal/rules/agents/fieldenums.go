package agents

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&fieldEnums{}) }

// fieldEnums warns when an agent's closed-enum frontmatter fields hold
// a value outside their documented set. Claude Code falls back to the
// default for unknown values without any runtime signal, so a typo'd
// permissionMode or effort silently changes behavior. Absent keys are
// fine — every enum field has a default.
type fieldEnums struct{}

func (*fieldEnums) ID() string                     { return "agents/field-enums" }
func (*fieldEnums) Category() string               { return categorySchema }
func (*fieldEnums) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*fieldEnums) DefaultOptions() map[string]any { return nil }
func (*fieldEnums) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindAgent}
}

func (*fieldEnums) HelpURI() string { return rules.DefaultHelpURI("agents/field-enums") }

// enumFields lists the closed-enum agent fields with their valid sets
// and the documented value order for messages (map iteration order is
// unstable). Isolation has a single documented value, expressed as a
// one-entry set for uniformity.
var enumFields = []struct {
	key   string
	valid map[string]struct{}
	want  string
}{
	{
		key:   "permissionMode",
		valid: artifact.AgentPermissionModes,
		want:  "default, acceptEdits, auto, dontAsk, bypassPermissions, plan, manual",
	},
	{
		key:   "effort",
		valid: artifact.AgentEffortLevels,
		want:  "low, medium, high, xhigh, max",
	},
	{
		key:   "color",
		valid: artifact.AgentColors,
		want:  "red, blue, green, yellow, purple, orange, pink, cyan",
	},
	{
		key:   "isolation",
		valid: map[string]struct{}{"worktree": {}},
		want:  "worktree",
	},
	{
		key:   "memory",
		valid: artifact.AgentMemoryScopes,
		want:  "user, project, local",
	},
}

func (r *fieldEnums) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	ag, ok := a.(*artifact.Agent)
	if !ok {
		return nil
	}
	values := map[string]string{
		"permissionMode": ag.PermissionMode,
		"effort":         ag.Effort,
		"color":          ag.Color,
		"isolation":      ag.Isolation,
		"memory":         ag.Memory,
	}
	var out []diag.Diagnostic
	for _, f := range enumFields {
		v := values[f.key]
		if v == "" {
			continue
		}
		if _, ok := f.valid[v]; ok {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID: r.ID(),
			Path:   ag.Path(),
			Range:  ag.Frontmatter.KeyRange(f.key),
			Message: fmt.Sprintf("%s %q is not a documented value; want one of: %s",
				f.key, v, f.want),
		})
	}
	return out
}
