// Package hooks holds rules that apply to Claude Code hook artifacts.
package hooks

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&eventNameKnown{}) }

// eventNameKnown errors when a hook declares an event name that is
// not in artifact.KnownHookEvents. A typo in `event` makes the hook
// silently inert, so the rule is error-severity.
type eventNameKnown struct{}

func (*eventNameKnown) ID() string                     { return "hooks/event-name-known" }
func (*eventNameKnown) Category() string               { return "schema" }
func (*eventNameKnown) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*eventNameKnown) DefaultOptions() map[string]any { return nil }
func (*eventNameKnown) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindHook}
}

func (r *eventNameKnown) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	h, ok := a.(*artifact.Hook)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	for i := range h.Entries {
		e := &h.Entries[i]
		if e.Event == "" || artifact.IsKnownHookEvent(e.Event) {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    h.Path(),
			Range:   e.EventRange,
			Message: fmt.Sprintf("unknown hook event %q", e.Event),
		})
	}
	return out
}
