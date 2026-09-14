package marketplace

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&pluginSourceValid{}) }

// pluginSourceValid errors when a plugins[] entry's source is missing
// or structurally incomplete for its shape: string sources must be
// non-empty; object sources must carry their kind's documented
// required fields (github → repo, url → url, git-subdir → url + path,
// npm → package, archive → url, command → command); a `sha` pin, when
// present, must be a full 40-char hex commit, and an archive's optional
// `sha256` a 64-char hex digest. Archive URLs must be HTTPS. The "does
// the path exist on disk" check is deliberately out of scope — rules are pure over the parsed artifact; filesystem
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
// fragments — one per missing or malformed documented requirement of
// its source shape. Each kind delegates so this stays a dispatch table
// rather than a nest of conditionals.
func shapeProblems(p *artifact.MarketplacePlugin) []string {
	src := &p.SourceInfo

	var out []string
	switch src.Kind {
	case artifact.SourceAbsent, artifact.SourceLocal, artifact.SourceExternalString:
		// String forms (and truly absent sources) need a non-empty
		// declaration; per-kind field checks don't apply.
		out = requireNonEmpty(p.Source, "is missing a non-empty source field")
	case artifact.SourceGitHub:
		out = requireNonEmpty(src.Repo, `github source requires a non-empty "repo" ("owner/repo")`)
	case artifact.SourceURL:
		out = requireNonEmpty(src.URL, `url source requires a non-empty "url"`)
	case artifact.SourceGitSubdir:
		out = append(
			requireNonEmpty(src.URL, `git-subdir source requires a non-empty "url"`),
			requireNonEmpty(src.Path, `git-subdir source requires a non-empty "path"`)...)
	case artifact.SourceNPM:
		out = requireNonEmpty(src.Package, `npm source requires a non-empty "package"`)
	case artifact.SourceArchive:
		out = archiveProblems(src)
	case artifact.SourceCommand:
		out = commandProblems(src)
	case artifact.SourceInvalid:
		out = []string{
			`has an unrecognized source; expected a relative path or an object whose ` +
				`"source" is one of github, url, git-subdir, npm, archive, command`,
		}
	}

	if src.SHA != "" && !isCommitSHA(src.SHA) {
		out = append(out, fmt.Sprintf("sha %q is not a full 40-character hex commit", src.SHA))
	}

	return out
}

// requireNonEmpty returns msg when value is empty, and nothing
// otherwise, so a one-field requirement reads as one line.
func requireNonEmpty(value, msg string) []string {
	if value == "" {
		return []string{msg}
	}

	return nil
}

// archiveProblems checks an archive source. The URL is required and
// must be HTTPS: an archive is unsigned code that Claude Code unpacks
// and runs, so a plain-HTTP fetch hands anyone on the path a plugin
// install. The optional sha256 is the only integrity check available,
// so a malformed one is worth failing on — it would otherwise look like
// a pin while pinning nothing.
func archiveProblems(src *artifact.MarketplaceSource) []string {
	var out []string
	switch {
	case src.URL == "":
		out = append(out, `archive source requires a non-empty "url"`)
	case !strings.HasPrefix(src.URL, "https://"):
		out = append(out, fmt.Sprintf("archive url %q must use https", src.URL))
	}
	if src.SHA256 != "" && !isSHA256(src.SHA256) {
		out = append(out, fmt.Sprintf("sha256 %q is not a 64-character hex digest", src.SHA256))
	}

	return out
}

// commandProblems checks a command source. The command itself is
// required; a declared timeout must be a positive integer, and is
// reported verbatim rather than silently read as zero.
func commandProblems(src *artifact.MarketplaceSource) []string {
	out := requireNonEmpty(strings.TrimSpace(src.Command), `command source requires a non-empty "command"`)
	if src.Timeout != "" {
		if n, err := strconv.Atoi(src.Timeout); err != nil || n <= 0 {
			out = append(out, fmt.Sprintf("command timeout %q is not a positive integer", src.Timeout))
		}
	}

	return out
}

// isSHA256 reports whether s is a 64-character hex digest.
func isSHA256(s string) bool { return len(s) == 64 && isHex(s) }

// isCommitSHA reports whether s is a full 40-character hex commit pin.
func isCommitSHA(s string) bool { return len(s) == 40 && isHex(s) }

// isHex reports whether every rune in s is a hex digit.
func isHex(s string) bool {
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}
