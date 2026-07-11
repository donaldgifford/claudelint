package marketplace

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&pluginSourceValid{}) }

// pluginSourceValid errors when a plugins[] entry's source is missing
// or structurally incomplete for its shape: string sources must be
// non-empty; object sources must carry their kind's documented
// required fields (github → repo, url → url, git-subdir → url + path,
// npm → package); a `sha` pin, when present, must be a full 40-char
// hex commit. The "does the path exist on disk" check is deliberately
// out of scope — rules are pure over the parsed artifact; filesystem
// validation belongs in a future engine-level pre-pass.
type pluginSourceValid struct{}

func (*pluginSourceValid) ID() string                     { return "marketplace/plugin-source-valid" }
func (*pluginSourceValid) Category() string               { return categorySchema }
func (*pluginSourceValid) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*pluginSourceValid) DefaultOptions() map[string]any { return nil }
func (*pluginSourceValid) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (*pluginSourceValid) HelpURI() string {
	return rules.DefaultHelpURI("marketplace/plugin-source-valid")
}

func (r *pluginSourceValid) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	for i := range m.Plugins {
		out = append(out, r.checkEntry(m.Path(), i, &m.Plugins[i])...)
	}
	return out
}

// checkEntry emits one diagnostic per structural problem in one
// plugins[] entry, anchored to the source's range when it has one.
func (r *pluginSourceValid) checkEntry(path string, i int, p *artifact.MarketplacePlugin) []diag.Diagnostic {
	problems := shapeProblems(p)
	if len(problems) == 0 {
		return nil
	}
	rng := p.SourceRange
	if rng.IsZero() {
		rng = p.NameRange
	}
	out := make([]diag.Diagnostic, 0, len(problems))
	for _, msg := range problems {
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    path,
			Range:   rng,
			Message: fmt.Sprintf("plugins[%d] %s", i, msg),
		})
	}
	return out
}

// shapeProblems lists an entry's structural problems as message
// fragments — one per missing documented requirement of its source
// shape.
func shapeProblems(p *artifact.MarketplacePlugin) []string {
	src := p.SourceInfo
	var out []string
	switch src.Kind {
	case artifact.SourceAbsent, artifact.SourceLocal, artifact.SourceExternalString:
		// String forms (and truly absent sources) need a non-empty
		// declaration; per-kind field checks don't apply.
		if p.Source == "" {
			out = append(out, "is missing a non-empty source field")
		}
	case artifact.SourceGitHub:
		if src.Repo == "" {
			out = append(out, `github source requires a non-empty "repo" ("owner/repo")`)
		}
	case artifact.SourceURL:
		if src.URL == "" {
			out = append(out, `url source requires a non-empty "url"`)
		}
	case artifact.SourceGitSubdir:
		if src.URL == "" {
			out = append(out, `git-subdir source requires a non-empty "url"`)
		}
		if src.Path == "" {
			out = append(out, `git-subdir source requires a non-empty "path"`)
		}
	case artifact.SourceNPM:
		if src.Package == "" {
			out = append(out, `npm source requires a non-empty "package"`)
		}
	case artifact.SourceInvalid:
		out = append(out,
			`has an unrecognized source; expected a relative path or an object whose `+
				`"source" is one of github, url, git-subdir, npm`)
	}
	if src.SHA != "" && !isCommitSHA(src.SHA) {
		out = append(out, fmt.Sprintf("sha %q is not a full 40-character hex commit", src.SHA))
	}
	return out
}

// isCommitSHA reports whether s is a full 40-character hex commit pin.
func isCommitSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}
