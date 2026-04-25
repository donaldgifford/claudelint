// Package artifact defines the typed value produced by a parser and
// consumed by rules. The engine sees artifacts through the Artifact
// interface; concrete struct types (ClaudeMD, Skill, Command, Agent,
// Hook, Plugin) arrive in phase 1.2 and each type-asserts on Kind()
// inside its Check function.
//
// The interface is deliberately narrow. A rule should only need Kind,
// Path, and Source to produce file-level diagnostics; typed access to
// frontmatter or JSON values happens through the concrete struct after
// a safe type assertion.
package artifact

// ArtifactKind identifies the Claude Code artifact a file represents.
// The engine groups rules by Kind so it can skip rules whose
// AppliesTo() does not include a given artifact's Kind.
type ArtifactKind string

// Canonical artifact kinds. Any new kind must also be added to
// discovery classification and AllKinds; the fingerprint test in phase
// 1.5 will catch a stale AllKinds via rule coverage drift.
const (
	// KindClaudeMD is a CLAUDE.md file at any depth.
	KindClaudeMD ArtifactKind = "claude_md"

	// KindSkill is .claude/skills/<name>/SKILL.md plus its companion
	// files.
	KindSkill ArtifactKind = "skill"

	// KindCommand is .claude/commands/*.md (slash-command definitions).
	KindCommand ArtifactKind = "command"

	// KindAgent is .claude/agents/*.md (subagent definitions).
	KindAgent ArtifactKind = "agent"

	// KindHook is hook configuration: .claude/hooks/*.json, or the
	// hooks stanza inside .claude/settings{,.local}.json.
	KindHook ArtifactKind = "hook"

	// KindPlugin is a plugin manifest (plugin.json or plugin.yaml).
	KindPlugin ArtifactKind = "plugin"

	// KindMarketplace is a plugin-marketplace manifest
	// (.claude-plugin/marketplace.json). It drives discovery: each
	// entry in its plugins[] array becomes a plugin root.
	KindMarketplace ArtifactKind = "marketplace"

	// KindMCPServer is a Model Context Protocol server declaration.
	// One artifact represents one server entry — either from a
	// standalone .mcp.json or a plugin.json's mcp.servers{} map.
	KindMCPServer ArtifactKind = "mcp_server"
)

// AllKinds returns the canonical list of artifact kinds in a stable
// order. It is the single source of truth that discovery, the engine,
// and `claudelint rules` iterate over; appending a new kind here is a
// deliberate act that shows up in a focused diff.
func AllKinds() []ArtifactKind {
	return []ArtifactKind{
		KindClaudeMD,
		KindSkill,
		KindCommand,
		KindAgent,
		KindHook,
		KindPlugin,
		KindMarketplace,
		KindMCPServer,
	}
}

// Artifact is the parsed, typed view of a file on disk. Parsers
// produce Artifacts; rules consume them. Rules see only the Artifact
// surface (plus their own Context) — never the filesystem.
//
// Kind identifies the concrete type a rule should type-assert to. Path
// is the repo-relative path used in diagnostics. Source is the raw
// bytes the parser consumed, kept so reporters can slice snippets out
// of it when rendering detail.
type Artifact interface {
	Kind() ArtifactKind
	Path() string
	Source() []byte
}
