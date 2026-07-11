package hooks

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&timeoutPresent{}) }

// timeoutPresent warns when a hook has no `timeout` declared. Claude
// Code applies documented per-type defaults (600 s for command/http/
// mcp_tool, 30 s for prompt, 60 s for agent), so this is a style
// nudge, not a hang guard: an explicit timeout fails fast in CI and
// records the author's latency expectation next to the hook.
type timeoutPresent struct{}

func (*timeoutPresent) ID() string                     { return "hooks/timeout-present" }
func (*timeoutPresent) Category() string               { return "content" }
func (*timeoutPresent) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*timeoutPresent) DefaultOptions() map[string]any { return nil }
func (*timeoutPresent) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindHook}
}

func (*timeoutPresent) HelpURI() string { return rules.DefaultHelpURI("hooks/timeout-present") }

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
		rng := e.CommandRange
		if rng.IsZero() {
			rng = e.TypeRange
		}
		typ := e.EffectiveType()
		out = append(out, diag.Diagnostic{
			RuleID: r.ID(),
			Path:   h.Path(),
			Range:  rng,
			Message: fmt.Sprintf(
				"hook has no explicit timeout; Claude Code defaults %s hooks to %d s — declare one to fail faster",
				typ, defaultTimeoutSecs(typ)),
		})
	}
	return out
}

// defaultTimeoutSecs is the documented default timeout for an
// effective hook type: 600 s for command/http/mcp_tool, 30 s for
// prompt, 60 s for agent.
func defaultTimeoutSecs(hookType string) int {
	switch hookType {
	case "prompt":
		return 30
	case "agent":
		return 60
	default:
		return 600
	}
}
