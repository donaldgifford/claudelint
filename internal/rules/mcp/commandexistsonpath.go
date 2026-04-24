package mcp

import (
	"strings"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&commandExistsOnPath{}) }

// commandExistsOnPath warns when an MCP server's `command` is a bare
// name (no slashes) that is not in a small allowlist of known MCP
// server runners. This is a typo catcher — `"uvv"` instead of
// `"uvx"` — not a PATH-resolution check; resolving against the host
// PATH would produce CI/laptop false positives.
//
// The rule fires at warning severity so users who intentionally rely
// on custom runners can bump it with `severity = "error"` in config
// once they have confidence in their setup.
type commandExistsOnPath struct{}

// knownRunners is the set of commands that typically launch MCP
// servers. It is not authoritative — anything absolute or containing
// a slash is skipped (the user knows what they're doing), and users
// can configure additional entries via options if we ever add that.
var knownRunners = map[string]struct{}{
	"uvx":     {},
	"uv":      {},
	"npx":     {},
	"bunx":    {},
	"pipx":    {},
	"python":  {},
	"python3": {},
	"node":    {},
	"docker":  {},
	"bash":    {},
	"sh":      {},
	"deno":    {},
	"go":      {},
}

func (*commandExistsOnPath) ID() string                     { return "mcp/command-exists-on-path" }
func (*commandExistsOnPath) Category() string               { return categorySchema }
func (*commandExistsOnPath) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*commandExistsOnPath) DefaultOptions() map[string]any { return nil }
func (*commandExistsOnPath) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (*commandExistsOnPath) HelpURI() string {
	return rules.DefaultHelpURI("mcp/command-exists-on-path")
}

func (r *commandExistsOnPath) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok || s.Command == "" {
		return nil
	}
	// Absolute or path-qualified commands are always fine.
	if strings.HasPrefix(s.Command, "/") || strings.ContainsRune(s.Command, '/') {
		return nil
	}
	if _, known := knownRunners[s.Command]; known {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    s.Path(),
		Range:   s.CommandRange,
		Message: "MCP server " + quoteName(s.Name) + " command " + quoteName(s.Command) + " is not a recognized runner; check for typos",
	}}
}
