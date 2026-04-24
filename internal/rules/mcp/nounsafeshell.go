package mcp

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&noUnsafeShell{}) }

// noUnsafeShell errors when an MCP server is invoked as
// `bash -c "..."` or `sh -c "..."`. That pattern loses the argv
// boundary between the server launcher and its arguments, making it
// trivial to inject shell metacharacters. MCP servers are expected
// to be long-running processes; there is no legitimate need for a
// -c shell wrap in a plugin manifest.
//
// Mirrors hooks/nounsafeshell from Phase 1.
type noUnsafeShell struct{}

func (*noUnsafeShell) ID() string                     { return "mcp/no-unsafe-shell" }
func (*noUnsafeShell) Category() string               { return categorySecurity }
func (*noUnsafeShell) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*noUnsafeShell) DefaultOptions() map[string]any { return nil }
func (*noUnsafeShell) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (*noUnsafeShell) HelpURI() string { return rules.DefaultHelpURI("mcp/no-unsafe-shell") }

func (r *noUnsafeShell) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok {
		return nil
	}
	if s.Command != "bash" && s.Command != "sh" {
		return nil
	}
	for _, arg := range s.Args {
		if arg != "-c" {
			continue
		}
		return []diag.Diagnostic{{
			RuleID:  r.ID(),
			Path:    s.Path(),
			Range:   s.CommandRange,
			Message: "MCP server " + quoteName(s.Name) + ` invokes "` + s.Command + ` -c" — avoid shell wrappers in server commands`,
		}}
	}
	return nil
}
