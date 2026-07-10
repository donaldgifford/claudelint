package artifact

import "testing"

func TestIsKnownTool(t *testing.T) {
	if !IsKnownTool("Bash") {
		t.Errorf("Bash should be a known tool")
	}
	if IsKnownTool("bash") {
		t.Errorf("tool names are case-sensitive; 'bash' must not match")
	}
	if IsKnownTool("") {
		t.Errorf("empty string is not a tool name")
	}
	if IsKnownTool("Unknown") {
		t.Errorf("Unknown should not be a known tool")
	}
}

func TestKnownToolsIncludesRenameAndAlias(t *testing.T) {
	// Task was renamed Agent in v2.1.63; both spellings stay valid.
	for _, name := range []string{"Agent", "Task", "Skill"} {
		if !IsKnownTool(name) {
			t.Errorf("IsKnownTool(%q) = false, want true", name)
		}
	}
}

func TestIsToolPattern(t *testing.T) {
	tests := []struct {
		name  string
		entry string
		want  bool
	}{
		{"mcp server pattern", "mcp__github", true},
		{"mcp tool pattern", "mcp__github__get_issue", true},
		{"mcp wildcard pattern", "mcp__github__*", true},
		{"bare mcp prefix", "mcp__", false},
		{"permission rule", "Bash(git add:*)", true},
		{"agent form", "Agent(reviewer)", true},
		{"mcp base permission rule", "mcp__github__get_issue(owner:*)", true},
		{"unknown base", "Frobnicate(x)", false},
		{"empty specifier", "Bash()", false},
		{"missing close paren", "Bash(git add:*", false},
		{"bare known name is not a pattern", "Bash", false},
		{"bare unknown name", "Frobnicate", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsToolPattern(tt.entry); got != tt.want {
				t.Errorf("IsToolPattern(%q) = %v, want %v", tt.entry, got, tt.want)
			}
		})
	}
}

func TestIsKnownHookEvent(t *testing.T) {
	if !IsKnownHookEvent("PreToolUse") {
		t.Errorf("PreToolUse should be known")
	}
	if IsKnownHookEvent("pre_tool_use") {
		t.Errorf("case and spelling matter")
	}
}

// TestKnownHookEventsMatchesReference pins the expansion to the hooks
// reference (30 events as of 2026-07). A count change means the docs
// moved — update the list, this test, and the rules-doc table together.
func TestKnownHookEventsMatchesReference(t *testing.T) {
	if got := len(KnownHookEvents); got != 30 {
		t.Errorf("len(KnownHookEvents) = %d, want 30", got)
	}
	for _, name := range []string{
		"Setup", "PermissionRequest", "SubagentStart", "PostToolBatch",
		"UserPromptExpansion", "WorktreeCreate", "ElicitationResult",
	} {
		if !IsKnownHookEvent(name) {
			t.Errorf("IsKnownHookEvent(%q) = false, want true", name)
		}
	}
	if IsKnownHookEvent("PermissionsRequest") {
		t.Errorf("near-miss spellings must stay unknown")
	}
}
