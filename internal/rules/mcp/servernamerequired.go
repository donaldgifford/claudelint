package mcp

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&serverNameRequired{}) }

// serverNameRequired errors when an MCP server entry has an empty
// name. Empty keys are legal JSON but meaningless for MCP — Claude
// Code needs a unique handle per server for UI and routing.
type serverNameRequired struct{}

func (*serverNameRequired) ID() string                     { return "mcp/server-name-required" }
func (*serverNameRequired) Category() string               { return categorySchema }
func (*serverNameRequired) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*serverNameRequired) DefaultOptions() map[string]any { return nil }
func (*serverNameRequired) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (r *serverNameRequired) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok || s.Name != "" {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    s.Path(),
		Message: "MCP server entry has an empty name",
	}}
}
