package mcp

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&transportDeprecated{}) }

// transportDeprecated emits an info notice for servers declaring the
// `sse` transport, which the docs mark deprecated in favor of `http`.
// Split from mcp/transport-known because the engine assigns one
// severity per rule and deprecation is advisory, not a mistake.
type transportDeprecated struct{}

func (*transportDeprecated) ID() string                     { return "mcp/transport-deprecated" }
func (*transportDeprecated) Category() string               { return categorySchema }
func (*transportDeprecated) DefaultSeverity() diag.Severity { return diag.SeverityInfo }
func (*transportDeprecated) DefaultOptions() map[string]any { return nil }
func (*transportDeprecated) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (*transportDeprecated) HelpURI() string {
	return rules.DefaultHelpURI("mcp/transport-deprecated")
}

func (r *transportDeprecated) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok || s.Transport != "sse" {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    s.Path(),
		Range:   s.TransportRange,
		Message: `transport "sse" is documented as deprecated; migrate to "http"`,
	}}
}
