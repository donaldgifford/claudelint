package artifact

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

// KnownHookEvents is the canonical list of Claude Code hook event
// names. As with KnownTools, adding an event here changes the ruleset
// fingerprint.
var KnownHookEvents = map[string]struct{}{
	"PreToolUse":       {},
	"PostToolUse":      {},
	"UserPromptSubmit": {},
	"Notification":     {},
	"Stop":             {},
	"SubagentStop":     {},
	"PreCompact":       {},
	"SessionStart":     {},
	"SessionEnd":       {},
}

// IsKnownHookEvent reports whether name is in the canonical hook
// event list.
func IsKnownHookEvent(name string) bool {
	_, ok := KnownHookEvents[name]
	return ok
}
