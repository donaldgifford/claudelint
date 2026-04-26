package artifact

import (
	"strings"
	"testing"
)

func TestParseHookDedicatedFile(t *testing.T) {
	src := []byte(`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          { "type": "command", "command": "echo ok", "timeout": 30 }
        ]
      }
    ]
  }
}`)
	h, perr := ParseHook(".claude/hooks/guard.json", src)
	if perr != nil {
		t.Fatalf("ParseHook = %v, want nil", perr)
	}
	if h.Embedded {
		t.Errorf("dedicated file should not be Embedded")
	}
	if len(h.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(h.Entries))
	}
	e := h.Entries[0]
	if e.Event != "PreToolUse" {
		t.Errorf("Event = %q, want PreToolUse", e.Event)
	}
	if e.Matcher != "Bash" {
		t.Errorf("Matcher = %q, want Bash", e.Matcher)
	}
	if e.Command != "echo ok" {
		t.Errorf("Command = %q, want echo ok", e.Command)
	}
	if e.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", e.Timeout)
	}
	if e.CommandRange.IsZero() {
		t.Errorf("CommandRange should be populated")
	}
}

// TestParseHookPluginNestedShape covers the regression behind
// issue #14: a plugin hooks/hooks.json with timeout declared per
// inner entry was previously read by the flat-shape parser, which
// looked for `timeout` at the top level and produced Timeout == 0.
func TestParseHookPluginNestedShape(t *testing.T) {
	src := []byte(`{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "bash do-thing.sh",
            "timeout": 60
          }
        ]
      }
    ]
  }
}`)
	h, perr := ParseHook("plugins/example/hooks/hooks.json", src)
	if perr != nil {
		t.Fatalf("ParseHook = %v, want nil", perr)
	}
	if h.Embedded {
		t.Errorf("plugin hooks.json should not be Embedded")
	}
	if len(h.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(h.Entries))
	}
	e := h.Entries[0]
	if e.Event != "Stop" {
		t.Errorf("Event = %q, want Stop", e.Event)
	}
	if e.Command != "bash do-thing.sh" {
		t.Errorf("Command = %q, want bash do-thing.sh", e.Command)
	}
	if e.Timeout != 60 {
		t.Errorf("Timeout = %d, want 60", e.Timeout)
	}
	if e.TimeoutRange.IsZero() {
		t.Errorf("TimeoutRange should be populated")
	}
}

// TestParseHookDedicatedFileMissingHooksKey asserts dedicated hook
// files (non-settings) fail loudly when they don't carry a "hooks"
// key — rejecting both legacy flat-shape files and any other unknown
// shape rather than silently producing an entry with Timeout == 0.
func TestParseHookDedicatedFileMissingHooksKey(t *testing.T) {
	flat := []byte(`{"event":"PreToolUse","matcher":"Bash","command":"echo ok","timeout":30}`)
	_, perr := ParseHook(".claude/hooks/legacy.json", flat)
	if perr == nil {
		t.Fatal("expected ParseError for flat-shape hook file")
	}
	if !strings.Contains(perr.Message, `"hooks" key`) {
		t.Errorf("message = %q, want mention of missing hooks key", perr.Message)
	}
}

func TestParseHookSettingsFile(t *testing.T) {
	src := []byte(`{
  "hooks": {
    "PreToolUse": [
      { "matcher": "Bash", "hooks": [{ "command": "echo a", "timeout": 5 }] },
      { "matcher": "Edit", "hooks": [{ "command": "echo b" }] }
    ],
    "Stop": [
      { "hooks": [{ "command": "echo stop" }] }
    ]
  }
}`)
	h, perr := ParseHook(".claude/settings.json", src)
	if perr != nil {
		t.Fatalf("ParseHook = %v, want nil", perr)
	}
	if !h.Embedded {
		t.Errorf("settings.json should be Embedded")
	}
	if len(h.Entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(h.Entries))
	}

	// Build a quick index for assertions.
	byCmd := make(map[string]HookEntry, len(h.Entries))
	for _, e := range h.Entries {
		byCmd[e.Command] = e
	}

	a := byCmd["echo a"]
	if a.Event != "PreToolUse" || a.Matcher != "Bash" || a.Timeout != 5 {
		t.Errorf("echo a entry = %+v", a)
	}
	b := byCmd["echo b"]
	if b.Event != "PreToolUse" || b.Matcher != "Edit" {
		t.Errorf("echo b entry = %+v", b)
	}
	stop := byCmd["echo stop"]
	if stop.Event != "Stop" {
		t.Errorf("echo stop entry = %+v", stop)
	}
}

func TestParseHookMissingHooksKeyIsOK(t *testing.T) {
	src := []byte(`{"other":"stuff"}`)
	h, perr := ParseHook(".claude/settings.json", src)
	if perr != nil {
		t.Fatalf("ParseHook = %v, want nil", perr)
	}
	if !h.Embedded {
		t.Errorf("should still be marked Embedded")
	}
	if len(h.Entries) != 0 {
		t.Errorf("no hooks → no entries, got %d", len(h.Entries))
	}
}

func TestParseHookInvalidJSON(t *testing.T) {
	_, perr := ParseHook(".claude/hooks/bad.json", []byte("not json at all"))
	if perr == nil {
		t.Fatal("expected ParseError")
	}
	if !strings.Contains(perr.Message, "JSON") {
		t.Errorf("message = %q, want contains 'JSON'", perr.Message)
	}
}

func TestParsePluginJSON(t *testing.T) {
	src := []byte(`{
  "name": "example",
  "version": "1.2.3",
  "description": "demo plugin",
  "commands": ["review","summarize"],
  "skills": ["writer"],
  "agents": []
}`)
	p, perr := ParsePlugin("plugin.json", src)
	if perr != nil {
		t.Fatalf("ParsePlugin = %v, want nil", perr)
	}
	if p.Name != "example" || p.Version != "1.2.3" {
		t.Errorf("name/version = %q/%q", p.Name, p.Version)
	}
	if len(p.Commands) != 2 || p.Commands[0] != "review" {
		t.Errorf("commands = %v", p.Commands)
	}
	if p.NameRange.IsZero() {
		t.Errorf("NameRange should be populated")
	}
	if p.Kind() != KindPlugin {
		t.Errorf("Kind = %q", p.Kind())
	}
}

func TestParsePluginYAMLIsNotSupported(t *testing.T) {
	_, perr := ParsePlugin("plugin.yaml", []byte("name: x\n"))
	if perr == nil {
		t.Fatal("expected ParseError for YAML manifest")
	}
	if !strings.Contains(perr.Message, "YAML") {
		t.Errorf("message = %q, want mention of YAML", perr.Message)
	}
}

func TestParsePluginInvalidJSON(t *testing.T) {
	_, perr := ParsePlugin("plugin.json", []byte("garbage"))
	if perr == nil {
		t.Fatal("expected ParseError")
	}
}
