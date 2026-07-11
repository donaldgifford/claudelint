package mcp

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&timeoutMinimum{}) }

// minTimeoutMS is the documented minimum tool-execution timeout.
const minTimeoutMS = 1000

// timeoutMinimum warns when a server declares a `timeout` below the
// documented 1000 ms minimum. The field is milliseconds; nearly every
// sub-1000 value in the wild is someone writing seconds ("timeout":
// 30), which Claude Code clamps or rejects — so the message leads
// with the unit hint. Absent (0) is fine.
type timeoutMinimum struct{}

func (*timeoutMinimum) ID() string                     { return "mcp/timeout-minimum" }
func (*timeoutMinimum) Category() string               { return categorySchema }
func (*timeoutMinimum) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*timeoutMinimum) DefaultOptions() map[string]any { return nil }
func (*timeoutMinimum) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (*timeoutMinimum) HelpURI() string { return rules.DefaultHelpURI("mcp/timeout-minimum") }

func (r *timeoutMinimum) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok || s.TimeoutMS <= 0 || s.TimeoutMS >= minTimeoutMS {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID: r.ID(),
		Path:   s.Path(),
		Range:  s.NameRange,
		Message: fmt.Sprintf(
			"timeout %d is below the documented 1000 ms minimum — the field is milliseconds, not seconds (did you mean %d?)",
			s.TimeoutMS, s.TimeoutMS*1000),
	}}
}
