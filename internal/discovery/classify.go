// Package discovery walks a repository, applies gitignore semantics,
// and classifies each remaining file into an artifact.ArtifactKind.
// Parsing happens later in internal/artifact — discovery only decides
// "is this file something claudelint cares about, and if so, what is
// it?"
package discovery

import (
	"path/filepath"
	"strings"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// subRule matches paths underneath a .claude/ directory. A path
// qualifies when its subpath below ".claude/" starts with prefix and
// (if suffix is non-empty) ends with suffix. Exact matches skip prefix
// checking by setting prefix to the full expected subpath and suffix
// to the empty string.
type subRule struct {
	prefix, suffix string
	exact          bool
	kind           artifact.ArtifactKind
}

// claudeSubRules is the table of path patterns that live below
// .claude/. Order matters — longer prefixes first so KindSkill wins
// over a future wildcard rule.
var claudeSubRules = []subRule{
	{prefix: "settings.json", exact: true, kind: artifact.KindHook},
	{prefix: "settings.local.json", exact: true, kind: artifact.KindHook},
	{prefix: "hooks/", suffix: ".json", kind: artifact.KindHook},
	{prefix: "commands/", suffix: ".md", kind: artifact.KindCommand},
	{prefix: "agents/", suffix: ".md", kind: artifact.KindAgent},
	{prefix: "skills/", suffix: "/SKILL.md", kind: artifact.KindSkill},
}

// Classify returns the ArtifactKind for a repo-relative path, or
// (empty, false) if the path is not a Claude Code artifact. The logic
// lives in one function so discovery and any future tooling that wants
// to answer "is this a skill?" share a single definition.
//
// Path must be slash-separated and relative (use filepath.ToSlash on
// the output of filepath.Rel). An absolute path or one starting with
// "../" is rejected — discovery is responsible for producing clean
// relative paths before calling this.
func Classify(relPath string) (artifact.ArtifactKind, bool) {
	if relPath == "" || relPath == "." {
		return "", false
	}
	if filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "..") {
		return "", false
	}

	p := filepath.ToSlash(relPath)
	base := baseName(p)

	if k, ok := classifyMarketplace(p); ok {
		return k, true
	}

	if k, ok := classifyRoot(base); ok {
		return k, true
	}

	// Paths under a .claude/ segment are the canonical repo layout.
	if claudeIdx := claudeSegmentIndex(p); claudeIdx >= 0 {
		return classifyClaudeSub(p[claudeIdx+len(".claude/"):])
	}
	// Fallback: plugin-distribution layouts put skills/, commands/,
	// agents/, and hooks/ directly at the plugin root (no .claude/
	// parent). Match the last plugin-style segment so repos that
	// happen to use those directory names incidentally still classify
	// correctly when the path shape matches.
	return classifyPluginLayout(p)
}

// classifyMarketplace matches `.claude-plugin/marketplace.json` at any
// depth. The directory prefix is load-bearing — `marketplace.json` on
// its own is not enough to classify as a marketplace manifest.
func classifyMarketplace(p string) (artifact.ArtifactKind, bool) {
	const needle = ".claude-plugin/marketplace.json"
	if p == needle || strings.HasSuffix(p, "/"+needle) {
		return artifact.KindMarketplace, true
	}
	return "", false
}

// classifyRoot handles files whose classification depends only on
// their basename (CLAUDE.md at any depth, plugin manifests, MCP
// server declarations).
func classifyRoot(base string) (artifact.ArtifactKind, bool) {
	switch base {
	case "CLAUDE.md":
		return artifact.KindClaudeMD, true
	case "plugin.json", "plugin.yaml", "plugin.yml":
		return artifact.KindPlugin, true
	case ".mcp.json":
		return artifact.KindMCPServer, true
	}
	return "", false
}

// classifyClaudeSub matches the portion of a path that lies below a
// .claude/ segment against the claudeSubRules table.
func classifyClaudeSub(sub string) (artifact.ArtifactKind, bool) {
	for _, r := range claudeSubRules {
		if r.exact {
			if sub == r.prefix {
				return r.kind, true
			}
			continue
		}
		if strings.HasPrefix(sub, r.prefix) && strings.HasSuffix(sub, r.suffix) {
			return r.kind, true
		}
	}
	return "", false
}

// classifyPluginLayout handles Claude plugin distribution layouts.
// Plugin tarballs root their content as `<plugin>/{skills,commands,
// agents,hooks}/...`, without a `.claude/` parent. For these we match
// on the last occurrence of the plugin-style segment so the path
// `go-development/2.0.1/skills/go/SKILL.md` still resolves to a
// KindSkill. Only the leaf shape matters; anything above the segment
// is treated as repo-local noise.
func classifyPluginLayout(p string) (artifact.ArtifactKind, bool) {
	for _, r := range claudeSubRules {
		if r.exact {
			continue
		}
		// Find the last occurrence of "/<prefix>" and treat the
		// remainder as the sub path. Leading match (no slash) also
		// counts — a plugin tarball extracted to its own directory
		// may place the kind directory at the very top.
		rest, ok := trailingSubPath(p, r.prefix)
		if !ok {
			continue
		}
		if strings.HasSuffix(rest, r.suffix) {
			return r.kind, true
		}
	}
	return "", false
}

// trailingSubPath returns (sub, true) when prefix appears in p either
// at the start or immediately after a slash, and sub is the text from
// prefix onward. When multiple occurrences match, the deepest one
// wins so nested plugin-in-plugin layouts still classify correctly.
func trailingSubPath(p, prefix string) (string, bool) {
	needle := "/" + prefix
	if idx := strings.LastIndex(p, needle); idx >= 0 {
		return p[idx+1:], true
	}
	if strings.HasPrefix(p, prefix) {
		return p, true
	}
	return "", false
}

// claudeSegmentIndex finds the index where the ".claude/" segment
// starts, or -1 if the path has no .claude segment. It tolerates both
// a leading segment (.claude/...) and a nested one (plugin/.claude/...).
func claudeSegmentIndex(p string) int {
	const needle = ".claude/"
	if strings.HasPrefix(p, needle) {
		return 0
	}
	if i := strings.Index(p, "/"+needle); i >= 0 {
		return i + 1
	}
	return -1
}

func baseName(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
