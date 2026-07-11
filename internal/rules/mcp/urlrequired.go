package mcp

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&urlRequired{}) }

// urlRequired errors when a remote-transport MCP server (http, sse,
// ws) has no `url` or an empty one — there is nothing to connect to
// without it. The stdio counterpart is mcp/command-required. Unknown
// transport values are out of scope; a dedicated transport-known rule
// covers those.
type urlRequired struct{}

func (*urlRequired) ID() string                     { return "mcp/url-required" }
func (*urlRequired) Category() string               { return categorySchema }
func (*urlRequired) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*urlRequired) DefaultOptions() map[string]any { return nil }
func (*urlRequired) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (*urlRequired) HelpURI() string { return rules.DefaultHelpURI("mcp/url-required") }

func (r *urlRequired) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok {
		return nil
	}
	transport := s.EffectiveTransport()
	if transport != "http" && transport != "sse" && transport != "ws" {
		return nil
	}
	if s.URL != "" {
		return nil
	}
	rng := s.TransportRange
	if rng.IsZero() {
		rng = s.NameRange
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    s.Path(),
		Range:   rng,
		Message: fmt.Sprintf("MCP server %s has no url (required for %s transport)", quoteName(s.Name), transport),
	}}
}
