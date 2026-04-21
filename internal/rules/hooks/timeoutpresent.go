package hooks

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&timeoutPresent{}) }

// timeoutPresent warns when a hook has no `timeout` declared. A
// missing timeout lets the hook stall the entire session if its
// command hangs. v1 cannot distinguish fast and slow commands, so
// the rule warns on every hook.
type timeoutPresent struct{}

func (*timeoutPresent) ID() string                     { return "hooks/timeout-present" }
func (*timeoutPresent) Category() string               { return "content" }
func (*timeoutPresent) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*timeoutPresent) DefaultOptions() map[string]any { return nil }
func (*timeoutPresent) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindHook}
}

func (r *timeoutPresent) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	h, ok := a.(*artifact.Hook)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	for i := range h.Entries {
		e := &h.Entries[i]
		if e.Timeout > 0 {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    h.Path(),
			Range:   e.CommandRange,
			Message: "hook has no timeout declared",
		})
	}
	return out
}
