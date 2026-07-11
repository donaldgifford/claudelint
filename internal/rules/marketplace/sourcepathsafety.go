package marketplace

import (
	"fmt"
	"slices"
	"strings"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&sourcePathSafety{}) }

// sourcePathSafety errors when a local plugin source path is unsafe:
// a relative source must start with "./" and must not contain ".."
// segments. Claude Code's validator rejects both, and a ".." path
// that slipped through would read outside the marketplace repo.
// When the manifest declares metadata.pluginRoot, bare sources
// ("formatter") are documented-valid and only the ".." check applies.
// Remote shapes (github/url/git-subdir/npm) are out of scope, as are
// absent sources (plugin-source-valid's findings).
type sourcePathSafety struct{}

func (*sourcePathSafety) ID() string                     { return "marketplace/source-path-safety" }
func (*sourcePathSafety) Category() string               { return categorySecurity }
func (*sourcePathSafety) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*sourcePathSafety) DefaultOptions() map[string]any { return nil }
func (*sourcePathSafety) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (*sourcePathSafety) HelpURI() string {
	return rules.DefaultHelpURI("marketplace/source-path-safety")
}

func (r *sourcePathSafety) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	for i := range m.Plugins {
		p := &m.Plugins[i]
		if p.SourceInfo.Kind != artifact.SourceLocal || p.Source == "" {
			continue
		}
		if problem := pathProblem(p.Source, m.PluginRoot != ""); problem != "" {
			out = append(out, diag.Diagnostic{
				RuleID:  r.ID(),
				Path:    m.Path(),
				Range:   p.SourceRange,
				Message: fmt.Sprintf("plugin source %q %s", p.Source, problem),
			})
		}
	}
	return out
}

// pathProblem describes why a local source path is unsafe, or ""
// when it is fine. hasPluginRoot relaxes the "./" prefix requirement
// — metadata.pluginRoot makes bare sources documented-valid.
func pathProblem(src string, hasPluginRoot bool) string {
	if slices.Contains(strings.Split(strings.TrimPrefix(src, "./"), "/"), "..") {
		return `contains a ".." segment — sources must stay inside the marketplace repo`
	}
	if !hasPluginRoot && !strings.HasPrefix(src, "./") {
		return `must start with "./" — the validator rejects bare relative paths`
	}
	return ""
}
