package artifact

import "strings"

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
