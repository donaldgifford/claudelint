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

	if k, ok := classifyRoot(base); ok {
		return k, true
	}

	claudeIdx := claudeSegmentIndex(p)
	if claudeIdx < 0 {
		return "", false
	}
	return classifyClaudeSub(p[claudeIdx+len(".claude/"):])
}

// classifyRoot handles files whose classification depends only on
// their basename (CLAUDE.md at any depth, plugin manifests).
func classifyRoot(base string) (artifact.ArtifactKind, bool) {
	switch base {
	case "CLAUDE.md":
		return artifact.KindClaudeMD, true
	case "plugin.json", "plugin.yaml", "plugin.yml":
		return artifact.KindPlugin, true
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
