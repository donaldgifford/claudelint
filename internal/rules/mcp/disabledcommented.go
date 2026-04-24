package mcp

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&disabledCommented{}) }

// disabledCommented emits an info-level diagnostic for a disabled
// MCP server whose manifest carries no obvious "why" context. JSON
// does not support comments, so the best we can surface at rule time
// is the absence of an adjacent `description` field — a plain
// convention that some marketplaces already follow for disabled
// entries.
//
// Purely advisory; meant to nudge authors toward self-documenting
// configurations.
type disabledCommented struct{}

func (*disabledCommented) ID() string                     { return "mcp/disabled-commented" }
func (*disabledCommented) Category() string               { return categoryStyle }
func (*disabledCommented) DefaultSeverity() diag.Severity { return diag.SeverityInfo }
func (*disabledCommented) DefaultOptions() map[string]any { return nil }
func (*disabledCommented) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (r *disabledCommented) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok || !s.Disabled {
		return nil
	}
	// `description` is not parsed today; checking against the Env map
	// or a DescriptionField is not available on the artifact yet.
	// For now, just surface that a disabled server exists — rule-aware
	// users can mute the rule or extend the artifact in a follow-up.
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    s.Path(),
		Range:   s.NameRange,
		Message: "MCP server " + quoteName(s.Name) + " is disabled — consider documenting why with a description field",
	}}
}
