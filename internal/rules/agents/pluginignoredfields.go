package agents

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&pluginIgnoredFields{}) }

// pluginIgnoredFields warns when a plugin-distributed agent declares
// frontmatter Claude Code documents as ignored for plugin subagents:
// permissionMode, mcpServers, and hooks. The fields do nothing there —
// dead config that reads as if it took effect. Project- and user-level
// agents declaring the same fields are fine and stay silent.
type pluginIgnoredFields struct{}

func (*pluginIgnoredFields) ID() string                     { return "agents/plugin-ignored-fields" }
func (*pluginIgnoredFields) Category() string               { return categoryContent }
func (*pluginIgnoredFields) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*pluginIgnoredFields) DefaultOptions() map[string]any { return nil }
func (*pluginIgnoredFields) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindAgent}
}

func (*pluginIgnoredFields) HelpURI() string {
	return rules.DefaultHelpURI("agents/plugin-ignored-fields")
}

func (r *pluginIgnoredFields) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	ag, ok := a.(*artifact.Agent)
	if !ok || !ag.PluginDistributed {
		return nil
	}
	declared := []struct {
		key string
		set bool
	}{
		{"permissionMode", ag.PermissionMode != ""},
		{"mcpServers", ag.HasMCPServers},
		{"hooks", ag.HasHooks},
	}
	var out []diag.Diagnostic
	for _, f := range declared {
		if !f.set {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID: r.ID(),
			Path:   ag.Path(),
			Range:  ag.Frontmatter.KeyRange(f.key),
			Message: fmt.Sprintf(
				"%q is ignored for plugin-distributed subagents — dead config", f.key),
		})
	}
	return out
}
