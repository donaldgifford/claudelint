package artifact

import "github.com/donaldgifford/claudelint/internal/diag"

// ClaudeMD is a CLAUDE.md file. Frontmatter is optional — most
// CLAUDE.md files are pure Markdown; when present it is parsed for
// consistency with skill/command/agent semantics.
type ClaudeMD struct {
	Base
	Frontmatter Frontmatter

	// Body is the Range of the Markdown body below the frontmatter
	// fence (or of the whole file when no frontmatter is present).
	Body diag.Range
}

// Kind implements Artifact.
func (*ClaudeMD) Kind() ArtifactKind { return KindClaudeMD }

// Skill is .claude/skills/<name>/SKILL.md plus the bag of companion
// files (references/, scripts/, templates/) that live alongside it.
type Skill struct {
	Base
	Frontmatter Frontmatter
	Body        diag.Range

	Name         string
	Description  string
	Model        string
	AllowedTools []string

	// Companions are the files indexed by the skill parser. Each
	// entry is a relative path within the skill directory plus a
	// coarse kind (references, scripts, templates, other).
	Companions []Companion
}

// Kind implements Artifact.
func (*Skill) Kind() ArtifactKind { return KindSkill }

// Companion is an indexed file alongside a SKILL.md. Phase 1.5 skill
// rules read Companions to reason about referenced scripts and asset
// sizes without re-walking the filesystem.
type Companion struct {
	// RelPath is slash-separated and relative to the skill directory.
	RelPath string
	// Kind is one of "references", "scripts", "templates", "other".
	Kind string
}

// Command is a slash-command definition (.claude/commands/*.md).
type Command struct {
	Base
	Frontmatter Frontmatter
	Body        diag.Range

	Description  string
	ArgumentHint string
	AllowedTools []string
}

// Kind implements Artifact.
func (*Command) Kind() ArtifactKind { return KindCommand }

// Agent is a subagent definition (.claude/agents/*.md).
type Agent struct {
	Base
	Frontmatter Frontmatter
	Body        diag.Range

	Name        string
	Description string
	Tools       []string
}

// Kind implements Artifact.
func (*Agent) Kind() ArtifactKind { return KindAgent }

// Hook is a Claude Code hook artifact: a settings file
// (.claude/settings{,.local}.json) carrying a "hooks" stanza, a
// plugin hooks/hooks.json, or a .claude/hooks/*.json file. Every
// shape uses the same nested layout — see ParseHook — and one file
// usually carries multiple entries (one per event × matcher × hook
// command), so the artifact is a container over []HookEntry.
//
// Embedded == true distinguishes settings files (hooks share a file
// with other Claude Code config) from dedicated hook files; rules
// that only apply to one shape can switch on it.
type Hook struct {
	Base

	// Embedded is true when the source file is a settings.json (the
	// hooks are reached via the "hooks" key alongside other Claude
	// Code config), false for dedicated hook files.
	Embedded bool

	// Entries is the flattened cross-product of events × matchers ×
	// hook commands. May be empty when a settings file carries no
	// hooks; a dedicated hook file with no entries fails parsing.
	Entries []HookEntry
}

// HookEntry is one individual hook command with its event, matcher,
// and timeout. Every field carries its parsed byte-offset range so
// rules can point diagnostics at the precise JSON value.
type HookEntry struct {
	// Event is the hook event name (PreToolUse, PostToolUse, Stop, …).
	Event      string
	EventRange diag.Range

	// Matcher is the optional matcher pattern applied to tool names.
	Matcher      string
	MatcherRange diag.Range

	// Command is the shell command the hook runs.
	Command      string
	CommandRange diag.Range

	// Timeout is the declared timeout in seconds. Zero means "not
	// declared"; rules use Timeout == 0 for hooks/timeout-present.
	Timeout      int
	TimeoutRange diag.Range
}

// Kind implements Artifact.
func (*Hook) Kind() ArtifactKind { return KindHook }

// Plugin is a plugin manifest (plugin.json or plugin.yaml). Fields
// mirror the public plugin manifest schema; ranges are populated for
// every parsed value so rules can point at the exact offending key.
type Plugin struct {
	Base

	Name         string
	NameRange    diag.Range
	Version      string
	VersionRange diag.Range
	Description  string

	Commands []string
	Skills   []string
	Agents   []string
}

// Kind implements Artifact.
func (*Plugin) Kind() ArtifactKind { return KindPlugin }

// Marketplace is a plugin-marketplace manifest
// (.claude-plugin/marketplace.json). It carries the manifest-level
// fields plus the parsed plugins[] entries; discovery reads Plugins
// to drive the walker into each local plugin root.
type Marketplace struct {
	Base

	Name         string
	NameRange    diag.Range
	Version      string
	VersionRange diag.Range
	Author       string
	AuthorRange  diag.Range

	// Plugins is the parsed plugins[] array, in manifest order.
	Plugins []MarketplacePlugin
}

// Kind implements Artifact.
func (*Marketplace) Kind() ArtifactKind { return KindMarketplace }

// MarketplaceSourceKind names the shape a plugins[].source value took.
// The docs define one string form (a ./-relative path inside the
// marketplace repo) and four object forms.
type MarketplaceSourceKind string

const (
	// SourceAbsent means the entry declared no source at all.
	SourceAbsent MarketplaceSourceKind = ""
	// SourceLocal is the documented string form: a relative path
	// inside the marketplace repo.
	SourceLocal MarketplaceSourceKind = "local"
	// SourceExternalString is the legacy string shorthand for a
	// remote source: github:owner/repo, a URL, or git@host:path.
	// The docs express these as object forms today.
	SourceExternalString MarketplaceSourceKind = "external-string"
	// SourceGitHub is {"source": "github", "repo": "owner/repo", ...}.
	SourceGitHub MarketplaceSourceKind = "github"
	// SourceURL is {"source": "url", "url": "https://...", ...}.
	SourceURL MarketplaceSourceKind = "url"
	// SourceGitSubdir is {"source": "git-subdir", "url": ..., "path": ...}.
	SourceGitSubdir MarketplaceSourceKind = "git-subdir"
	// SourceNPM is {"source": "npm", "package": "@scope/name", ...}.
	SourceNPM MarketplaceSourceKind = "npm"
	// SourceInvalid is an object whose source discriminator is missing
	// or not one of the documented kinds.
	SourceInvalid MarketplaceSourceKind = "invalid"
)

// MarketplaceSource is the typed view of a plugins[].source value.
// Kind tells which shape was used; only that shape's fields are set.
type MarketplaceSource struct {
	Kind MarketplaceSourceKind

	// Repo is owner/repo (github kind).
	Repo string
	// URL is the git URL (url and git-subdir kinds).
	URL string
	// Path is the subdirectory inside the repo (git-subdir kind).
	Path string
	// Ref is a branch or tag; SHA is a full 40-char commit pin. Git
	// kinds only; SHA wins when both are set.
	Ref string
	SHA string
	// Package, Version, and Registry describe the npm kind.
	Package  string
	Version  string
	Registry string
}

// MarketplacePlugin is one entry in a marketplace manifest's plugins[]
// array. Resolved is the repo-relative path for local sources; it is
// the empty string for every remote shape.
type MarketplacePlugin struct {
	// Name is the plugins[].name field verbatim.
	Name      string
	NameRange diag.Range

	// Source is the plugins[].source string verbatim ("./",
	// "./plugins/foo", "github:owner/repo", ...). Empty when the
	// entry used an object source — see SourceInfo.
	Source      string
	SourceRange diag.Range

	// SourceInfo is the typed classification of the source, covering
	// both string and object shapes. SourceInfo.Kind == SourceAbsent
	// means the entry had no source field.
	SourceInfo MarketplaceSource

	// Resolved is the repo-relative path the source resolves to, or
	// "" if the source is remote or cannot be resolved. Always
	// slash-separated.
	Resolved string
}

// MCPServer is one MCP (Model Context Protocol) server declaration.
// It is the artifact unit even when many servers live in one file:
// one MCPServer per map entry in `.mcp.json`'s servers{} or in a
// plugin.json's mcp.servers{}. Per-entry artifacts let rules attach
// diagnostics to individual servers with precise byte ranges, the
// same approach Phase 1 used for hook entries.
type MCPServer struct {
	Base

	// Name is the map-key under servers{}.
	Name      string
	NameRange diag.Range

	// Command is the executable the server runs (typically a
	// language runner: uvx, npx, bunx, etc.). Required for the stdio
	// transport only.
	Command      string
	CommandRange diag.Range

	// Args is the argv passed to Command.
	Args []string

	// Env is the per-server environment map.
	Env map[string]string

	// Transport is the declared `type` key: stdio, http, sse
	// (deprecated), or ws. Empty when the entry omits type — use
	// EffectiveTransport for the documented stdio default. Rules that
	// distinguish declared from defaulted read this directly.
	Transport      string
	TransportRange diag.Range

	// URL is the remote endpoint, required for http/sse/ws transports.
	URL      string
	URLRange diag.Range

	// Headers are HTTP request headers for remote transports.
	Headers map[string]string

	// HeadersHelper is the command that generates dynamic headers at
	// connection time (http transport).
	HeadersHelper string

	// TimeoutMS is the per-server tool-execution timeout in
	// milliseconds; 0 means not declared. The documented minimum is
	// 1000.
	TimeoutMS int64

	// AlwaysLoad mirrors the alwaysLoad startup flag.
	AlwaysLoad bool

	// HasOAuth is true when an oauth{} object is present. Its fields
	// are not modeled yet — IMPL-0004 Phase 1 parses presence only.
	HasOAuth bool

	// Disabled mirrors the optional disabled flag — disabled servers
	// still parse but rules can choose to skip them.
	Disabled bool

	// Embedded is true when the server came from a plugin's
	// plugin.json (mcp.servers{}) rather than a standalone .mcp.json.
	// Some rules apply to only one context.
	Embedded bool

	// LegacyServersKey is true when a standalone .mcp.json declared
	// this server under the deprecated top-level servers{} key. The
	// docs standardized on mcpServers{} (IMPL-0004 OQ1); when both
	// keys are present mcpServers wins and this stays false. A rule
	// surfaces the deprecation as an info diagnostic.
	LegacyServersKey bool
}

// Kind implements Artifact.
func (*MCPServer) Kind() ArtifactKind { return KindMCPServer }

// EffectiveTransport returns the transport the server will use: the
// declared type, or the documented stdio default when absent.
func (s *MCPServer) EffectiveTransport() string {
	if s.Transport == "" {
		return "stdio"
	}
	return s.Transport
}

// Compile-time proof that every concrete type satisfies Artifact.
var (
	_ Artifact = (*ClaudeMD)(nil)
	_ Artifact = (*Skill)(nil)
	_ Artifact = (*Command)(nil)
	_ Artifact = (*Agent)(nil)
	_ Artifact = (*Hook)(nil)
	_ Artifact = (*Plugin)(nil)
	_ Artifact = (*Marketplace)(nil)
	_ Artifact = (*MCPServer)(nil)
)
