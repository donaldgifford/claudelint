package artifact

import (
	"regexp"
	"strings"
)

// KnownTools is the canonical list of built-in Claude Code tools a
// rule set or hook can reference. The list is maintained alongside
// public Claude Code documentation; adding a new tool here is a
// deliberate act that shows up in a focused diff and flips the
// ruleset fingerprint.
//
// Rules that need to validate tool names (commands/allowed-tools-known,
// hooks/event-name-known, etc.) read from this package rather than
// defining their own lists, so there is one source of truth.
var KnownTools = map[string]struct{}{
	"Agent":           {}, // renamed from Task in v2.1.63; both remain valid
	"AskUserQuestion": {},
	"Bash":            {},
	"BashOutput":      {},
	"Edit":            {},
	"ExitPlanMode":    {},
	"Glob":            {},
	"Grep":            {},
	"KillShell":       {},
	"MultiEdit":       {},
	"NotebookEdit":    {},
	"Read":            {},
	"Skill":           {},
	"Task":            {},
	"TodoWrite":       {},
	"WebFetch":        {},
	"WebSearch":       {},
	"Write":           {},
}

// IsKnownTool reports whether name is in the canonical tool list.
// Case-sensitive: "bash" is NOT a known tool; the spelling must match
// exactly what Claude Code's allowed-tools expects.
func IsKnownTool(name string) bool {
	_, ok := KnownTools[name]
	return ok
}

// IsToolPattern reports whether entry is a structurally valid tool
// reference that is not a bare tool name:
//
//   - mcp__<server> — every tool an MCP server exposes
//   - mcp__<server>__<tool> — one MCP tool (trailing wildcard allowed)
//   - Tool(specifier) — permission-rule form such as Bash(git add:*)
//     or Agent(reviewer), valid when the base before "(" is a known
//     tool or an MCP pattern and the specifier is non-empty
//
// Bare names remain IsKnownTool's job; rules validating tool lists
// accept an entry when either predicate holds.
func IsToolPattern(entry string) bool {
	if isMCPToolPattern(entry) {
		return true
	}
	open := strings.IndexByte(entry, '(')
	if open <= 0 || !strings.HasSuffix(entry, ")") {
		return false
	}
	if specifier := entry[open+1 : len(entry)-1]; specifier == "" {
		return false
	}
	base := entry[:open]
	return IsKnownTool(base) || isMCPToolPattern(base)
}

// isMCPToolPattern reports whether name references MCP-server tools:
// mcp__<server> or mcp__<server>__<tool>. Anything non-empty after
// the prefix counts — MCP server and tool names are user-defined, so
// there is no canonical list to check against.
func isMCPToolPattern(name string) bool {
	rest, ok := strings.CutPrefix(name, "mcp__")
	return ok && rest != ""
}

// KnownModelAliases are the documented model alias values accepted by
// agent, skill, and command `model` frontmatter.
var KnownModelAliases = map[string]struct{}{
	"sonnet": {},
	"opus":   {},
	"haiku":  {},
	"fable":  {},
}

// modelIDPattern matches full model IDs like claude-sonnet-5 or
// claude-opus-4-8.
var modelIDPattern = regexp.MustCompile(`^claude-[a-z0-9-]+$`)

// IsValidModelRef reports whether v is a documented model reference:
// an alias from KnownModelAliases, "inherit", or a full model ID.
// The empty string is the caller's concern — an absent model key
// means inherit, which is valid, so rules skip it before calling.
func IsValidModelRef(v string) bool {
	if _, ok := KnownModelAliases[v]; ok {
		return true
	}
	return v == "inherit" || modelIDPattern.MatchString(v)
}

// Agent frontmatter enum sets, mirroring the subagents reference
// (2026-07). agents/field-enums validates membership; isolation has a
// single documented value ("worktree") and is checked directly.
var (
	// AgentPermissionModes includes "manual", the documented alias
	// for "default" (Claude Code v2.1.200+).
	AgentPermissionModes = map[string]struct{}{
		"default":           {},
		"acceptEdits":       {},
		"auto":              {},
		"dontAsk":           {},
		"bypassPermissions": {},
		"plan":              {},
		"manual":            {},
	}

	AgentEffortLevels = map[string]struct{}{
		"low":    {},
		"medium": {},
		"high":   {},
		"xhigh":  {},
		"max":    {},
	}

	AgentColors = map[string]struct{}{
		"red":    {},
		"blue":   {},
		"green":  {},
		"yellow": {},
		"purple": {},
		"orange": {},
		"pink":   {},
		"cyan":   {},
	}

	AgentMemoryScopes = map[string]struct{}{
		"user":    {},
		"project": {},
		"local":   {},
	}
)

// KnownHookEvents is the canonical list of Claude Code hook event
// names, mirroring the hooks reference
// (https://code.claude.com/docs/en/hooks) — 30 events as of 2026-07.
// As with KnownTools, adding an event here changes the ruleset
// fingerprint. The full table with lifecycle groupings lives in the
// rules doc alongside hooks/event-name-known.
var KnownHookEvents = map[string]struct{}{
	"ConfigChange":        {},
	"CwdChanged":          {},
	"Elicitation":         {},
	"ElicitationResult":   {},
	"FileChanged":         {},
	"InstructionsLoaded":  {},
	"MessageDisplay":      {},
	"Notification":        {},
	"PermissionDenied":    {},
	"PermissionRequest":   {},
	"PostCompact":         {},
	"PostToolBatch":       {},
	"PostToolUse":         {},
	"PostToolUseFailure":  {},
	"PreCompact":          {},
	"PreToolUse":          {},
	"SessionEnd":          {},
	"SessionStart":        {},
	"Setup":               {},
	"Stop":                {},
	"StopFailure":         {},
	"SubagentStart":       {},
	"SubagentStop":        {},
	"TaskCompleted":       {},
	"TaskCreated":         {},
	"TeammateIdle":        {},
	"UserPromptExpansion": {},
	"UserPromptSubmit":    {},
	"WorktreeCreate":      {},
	"WorktreeRemove":      {},
}

// IsKnownHookEvent reports whether name is in the canonical hook
// event list.
func IsKnownHookEvent(name string) bool {
	_, ok := KnownHookEvents[name]
	return ok
}

// SuggestHookEvent returns the known hook event matching name
// case-insensitively, for did-you-mean diagnostics on casing typos —
// the most common way a hook goes silently inert ("PretoolUse").
// ok is false when no known event matches.
func SuggestHookEvent(name string) (suggestion string, ok bool) {
	for event := range KnownHookEvents {
		if strings.EqualFold(event, name) {
			return event, true
		}
	}
	return "", false
}

// KnownHookTypes mirrors the five documented hook type values
// (2026-07). An absent `type` defaults to "command"
// (HookEntry.EffectiveType), so rules only validate declared values.
var KnownHookTypes = map[string]struct{}{
	"command":  {},
	"http":     {},
	"mcp_tool": {},
	"prompt":   {},
	"agent":    {},
}

// IsKnownHookType reports whether name is a documented hook type.
func IsKnownHookType(name string) bool {
	_, ok := KnownHookTypes[name]
	return ok
}
