// Package mcp holds rules that apply to MCP server declarations
// (KindMCPServer), sourced from either a standalone .mcp.json or a
// plugin.json's mcp.servers{} stanza.
package mcp

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// Shared category constants. Declared once per package so the rules
// below can reference them without triggering goconst.
const (
	categorySchema   = "schema"
	categoryStyle    = "style"
	categorySecurity = "security"
)

func init() { rules.Register(&commandRequired{}) }

// commandRequired errors when an MCP server has no `command` or an
// empty one — the server cannot be launched without it.
type commandRequired struct{}

func (*commandRequired) ID() string                     { return "mcp/command-required" }
func (*commandRequired) Category() string               { return categorySchema }
func (*commandRequired) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*commandRequired) DefaultOptions() map[string]any { return nil }
func (*commandRequired) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (*commandRequired) HelpURI() string { return rules.DefaultHelpURI("mcp/command-required") }

func (r *commandRequired) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok {
		return nil
	}
	if s.Command != "" {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    s.Path(),
		Range:   s.NameRange,
		Message: "MCP server " + quoteName(s.Name) + " has no command",
	}}
}

// quoteName renders a server's map-key for inclusion in diagnostic
// messages. Extracted to keep message construction uniform across
// this package's rules.
func quoteName(n string) string {
	if n == "" {
		return `"(unnamed)"`
	}
	return `"` + n + `"`
}
