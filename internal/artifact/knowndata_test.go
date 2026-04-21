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

func TestIsKnownHookEvent(t *testing.T) {
	if !IsKnownHookEvent("PreToolUse") {
		t.Errorf("PreToolUse should be known")
	}
	if IsKnownHookEvent("pre_tool_use") {
		t.Errorf("case and spelling matter")
	}
}
