package mcp

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&transportKnown{}) }

// transportKnown warns when a server declares a `type` outside the
// four documented transports: stdio, http, sse, ws. Claude Code
// cannot connect over an unknown transport, so the server silently
// never loads. Omitting the key is fine — it defaults to stdio.
// The sse-is-deprecated notice is transportDeprecated's job (info
// severity; the engine assigns one severity per rule).
type transportKnown struct{}

func (*transportKnown) ID() string                     { return "mcp/transport-known" }
func (*transportKnown) Category() string               { return categorySchema }
func (*transportKnown) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*transportKnown) DefaultOptions() map[string]any { return nil }
func (*transportKnown) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (*transportKnown) HelpURI() string { return rules.DefaultHelpURI("mcp/transport-known") }

// knownTransports mirrors the documented `type` values (2026-07).
var knownTransports = map[string]struct{}{
	"stdio": {},
	"http":  {},
	"sse":   {},
	"ws":    {},
}

func (r *transportKnown) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok || s.Transport == "" {
		return nil
	}
	if _, known := knownTransports[s.Transport]; known {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID: r.ID(),
		Path:   s.Path(),
		Range:  s.TransportRange,
		Message: fmt.Sprintf(
			"unknown transport %q; want stdio, http, sse, or ws", s.Transport),
	}}
}
