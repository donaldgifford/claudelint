package mcp

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&legacyServersKey{}) }

// legacyServersKey emits an info diagnostic when a .mcp.json file
// declares its servers under the deprecated top-level `servers` key.
// The docs standardized on `mcpServers`; both are accepted for now
// (OQ1), with `servers` support scheduled to drop at the next major
// ruleset revision.
//
// Every server parsed from a legacy file carries the flag, so the
// rule emits one identical file-level diagnostic per server and
// relies on the engine's exact-duplicate dedupe to collapse them to
// one notice per file.
type legacyServersKey struct{}

func (*legacyServersKey) ID() string                     { return "mcp/legacy-servers-key" }
func (*legacyServersKey) Category() string               { return categorySchema }
func (*legacyServersKey) DefaultSeverity() diag.Severity { return diag.SeverityInfo }
func (*legacyServersKey) DefaultOptions() map[string]any { return nil }
func (*legacyServersKey) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (*legacyServersKey) HelpURI() string { return rules.DefaultHelpURI("mcp/legacy-servers-key") }

func (r *legacyServersKey) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok || !s.LegacyServersKey {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    s.Path(),
		Message: `.mcp.json uses the deprecated "servers" key; rename it to "mcpServers"`,
	}}
}
