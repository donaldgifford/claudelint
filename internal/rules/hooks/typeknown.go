package hooks

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&typeKnown{}) }

// Shared strings, extracted once three rules in the package
// reference them (goconst).
const (
	categorySchema = "schema"
	typeCommand    = "command"
)

// typeKnown errors when a hook entry declares a `type` outside the
// five documented values. Claude Code treats an unknown type as
// misconfiguration, so the hook never fires. An absent `type` is fine
// — it defaults to "command" (HookEntry.EffectiveType).
type typeKnown struct{}

func (*typeKnown) ID() string                     { return "hooks/type-known" }
func (*typeKnown) Category() string               { return categorySchema }
func (*typeKnown) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*typeKnown) DefaultOptions() map[string]any { return nil }
func (*typeKnown) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindHook}
}

func (*typeKnown) HelpURI() string { return rules.DefaultHelpURI("hooks/type-known") }

func (r *typeKnown) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	h, ok := a.(*artifact.Hook)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	for i := range h.Entries {
		e := &h.Entries[i]
		if e.Type == "" || artifact.IsKnownHookType(e.Type) {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID: r.ID(),
			Path:   h.Path(),
			Range:  e.TypeRange,
			Message: fmt.Sprintf(
				"unknown hook type %q; want command, http, mcp_tool, prompt, or agent",
				e.Type),
		})
	}
	return out
}
