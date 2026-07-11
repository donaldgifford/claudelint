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
// Skills and slash commands share one frontmatter model in the docs,
// so the typed fields below mirror Command's.
type Skill struct {
	Base
	Frontmatter Frontmatter
	Body        diag.Range

	Name        string
	Description string
	Model       string
	// WhenToUse supplements Description for the model's
	// should-I-invoke-this decision (frontmatter key when_to_use).
	WhenToUse string
	// Context is the execution context; the documented value is
	// "fork", which pairs with Agent.
	Context string
	// Agent names the subagent that runs the skill when Context is
	// "fork".
	Agent           string
	AllowedTools    []string
	DisallowedTools []string
	// DisableModelInvocation mirrors the declared bool; absent means
	// false (the runtime default — the model may invoke the skill).
	DisableModelInvocation bool
	// UserInvocable is nil when the key is absent; the runtime
	// defaults to true.
	UserInvocable *bool

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
// Commands share the skill frontmatter model — see Skill for field
// semantics; ArgumentHint is the one command-only field.
type Command struct {
	Base
	Frontmatter Frontmatter
	Body        diag.Range

	Description     string
	ArgumentHint    string
	Model           string
	WhenToUse       string
	Context         string
	Agent           string
	AllowedTools    []string
	DisallowedTools []string
	// DisableModelInvocation mirrors the declared bool; absent means
	// false (the runtime default).
	DisableModelInvocation bool
	// UserInvocable is nil when the key is absent; the runtime
	// defaults to true.
	UserInvocable *bool
}

// Kind implements Artifact.
func (*Command) Kind() ArtifactKind { return KindCommand }

// Agent is a subagent definition (.claude/agents/*.md). The typed
// fields mirror the documented 16-field subagent frontmatter spec
// (DESIGN-0005 §1); every declared key's range is available via
// Frontmatter.KeyRange.
type Agent struct {
	Base
	Frontmatter Frontmatter
	Body        diag.Range

	Name            string
	Description     string
	Tools           []string
	DisallowedTools []string
	// Model is the raw declared value; absent means inherit.
	Model          string
	PermissionMode string
	// MaxTurns is 0 when absent or non-numeric; presence is
	// distinguishable via Frontmatter.KeyRange("maxTurns").
	MaxTurns int64
	// Skills is the list of skills preloaded into the subagent's
	// context — skill names, not tool syntax, so no splitting.
	Skills []string
	// HasMCPServers / HasHooks record presence only; the value
	// shapes are not modeled (DESIGN-0005 non-goal).
	HasMCPServers bool
	HasHooks      bool
	Memory        string
	// Background false covers both declared-false and absent
	// ("Claude chooses"); rules that need the distinction can check
	// the key range.
	Background    bool
	Effort        string
	Isolation     string
	Color         string
	InitialPrompt string

	// PluginDistributed is set by discovery (not the parser) when the
	// agent file lives under a root containing
	// .claude-plugin/plugin.json. Claude Code ignores permissionMode,
	// mcpServers, and hooks on plugin-distributed subagents.
	PluginDistributed bool
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

// HookEntry is one individual hook with its event, matcher, type, and
// per-type fields. Every field carries its parsed byte-offset range so
// rules can point diagnostics at the precise JSON value.
type HookEntry struct {
	// Event is the hook event name (PreToolUse, PostToolUse, Stop, …).
	Event      string
	EventRange diag.Range

	// Matcher is the optional matcher pattern applied to tool names.
	Matcher      string
	MatcherRange diag.Range

	// Type is the declared hook type: command, http, mcp_tool, prompt,
	// or agent. Empty when omitted — use EffectiveType for the
	// documented command default. Rules that distinguish declared from
	// defaulted (hooks/type-known) read this directly.
	Type      string
	TypeRange diag.Range

	// Command is the shell command a command-type hook runs.
	Command      string
	CommandRange diag.Range

	// URL is the POST endpoint of an http-type hook.
	URL      string
	URLRange diag.Range

	// Server and Tool identify an mcp_tool-type hook's target.
	Server string
	Tool   string

	// Prompt is the prompt/agent-type hook's instruction text.
	Prompt string

	// ExecForm is true when the entry declares args[] — the command
	// is spawned directly with no shell, so shell-smell heuristics
	// (hooks/no-unsafe-shell) do not apply.
	ExecForm bool

	// Async mirrors the async background-execution flag.
	Async bool

	// Shell is the declared shell for command hooks: bash or
	// powershell. Empty when omitted (bash default).
	Shell string

	// Timeout is the declared timeout in seconds. Zero means "not
	// declared"; rules use Timeout == 0 for hooks/timeout-present.
	Timeout      int
	TimeoutRange diag.Range
}

// EffectiveType returns the hook type the entry will run as: the
// declared type, or the documented command default when absent.
func (e *HookEntry) EffectiveType() string {
	if e.Type == "" {
		return "command"
	}
	return e.Type
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

	// OwnerName/OwnerEmail are the documented root owner{} object.
	// The docs require owner.name; Author above remains the merged
	// legacy view (top-level author string, falling back to
	// owner.name) that existing rules consume.
	OwnerName  string
	OwnerRange diag.Range
	OwnerEmail string

	// Renames maps former plugin names to current ones for automatic
	// migration. A JSON null target (plugin removed) parses as "" —
	// plugin names cannot legitimately be empty. Nil when absent.
	Renames map[string]string

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
