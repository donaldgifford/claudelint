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

// Hook is a Claude Code hook declaration, either from a dedicated
// .claude/hooks/*.json file or embedded in the "hooks" key of
// .claude/settings{,.local}.json. Fields reflect the public Claude
// Code hook schema; ranges track byte offsets for every parsed value.
type Hook struct {
	Base

	// Event is the hook event name (PreToolUse, PostToolUse, Stop, …).
	Event      string
	EventRange diag.Range

	// Matcher is the optional matcher pattern applied to tool names.
	Matcher      string
	MatcherRange diag.Range

	// Command is the shell command the hook runs.
	Command      string
	CommandRange diag.Range

	// Timeout is parsed as a raw integer of seconds; zero means
	// "not declared". Rules check Timeout == 0 for the
	// hooks/timeout-present lint.
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

// Compile-time proof that every concrete type satisfies Artifact.
var (
	_ Artifact = (*ClaudeMD)(nil)
	_ Artifact = (*Skill)(nil)
	_ Artifact = (*Command)(nil)
	_ Artifact = (*Agent)(nil)
	_ Artifact = (*Hook)(nil)
	_ Artifact = (*Plugin)(nil)
)
