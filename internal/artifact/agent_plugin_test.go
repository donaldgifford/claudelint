package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

// writePluginLayout creates root/.claude-plugin/plugin.json plus an
// agent file at the given relative path, returning the agent's
// absolute path.
func writePluginLayout(t *testing.T, root, agentRel string) string {
	t.Helper()
	manifestDir := filepath.Join(root, ".claude-plugin")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	agentPath := filepath.Join(root, agentRel)
	if err := os.MkdirAll(filepath.Dir(agentPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentPath, []byte("---\nname: a\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return agentPath
}

func TestMarkPluginDistributed(t *testing.T) {
	t.Run("agent under plugin root is marked", func(t *testing.T) {
		root := t.TempDir()
		agentPath := writePluginLayout(t, root, filepath.Join("agents", "a.md"))
		var a Agent
		MarkPluginDistributed(&a, agentPath, root)
		if !a.PluginDistributed {
			t.Error("MarkPluginDistributed(agents/a.md under plugin root) = false, want true")
		}
	})

	t.Run("deeply nested agent is marked", func(t *testing.T) {
		root := t.TempDir()
		agentPath := writePluginLayout(t, root,
			filepath.Join("plugins", "tools", "agents", "deep", "a.md"))
		var a Agent
		MarkPluginDistributed(&a, agentPath, root)
		if !a.PluginDistributed {
			t.Error("deeply nested agent under plugin root not marked")
		}
	})

	t.Run("project-level agent stays unmarked", func(t *testing.T) {
		root := t.TempDir()
		agentPath := filepath.Join(root, ".claude", "agents", "a.md")
		if err := os.MkdirAll(filepath.Dir(agentPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(agentPath, []byte("---\nname: a\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var a Agent
		MarkPluginDistributed(&a, agentPath, root)
		if a.PluginDistributed {
			t.Error("project-level agent marked as plugin-distributed")
		}
	})

	t.Run("walk stops at root without escaping", func(t *testing.T) {
		outer := t.TempDir()
		// Manifest lives ABOVE the scan root — must not be found.
		if err := os.MkdirAll(filepath.Join(outer, ".claude-plugin"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(outer, ".claude-plugin", "plugin.json"), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
		root := filepath.Join(outer, "repo")
		agentPath := filepath.Join(root, "agents", "a.md")
		if err := os.MkdirAll(filepath.Dir(agentPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(agentPath, []byte("---\nname: a\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var a Agent
		MarkPluginDistributed(&a, agentPath, root)
		if a.PluginDistributed {
			t.Error("walk escaped the scan root and found an outer manifest")
		}
	})

	t.Run("empty root walks up bounded", func(t *testing.T) {
		root := t.TempDir()
		agentPath := writePluginLayout(t, root, filepath.Join("agents", "a.md"))
		var a Agent
		MarkPluginDistributed(&a, agentPath, "")
		if !a.PluginDistributed {
			t.Error("empty root should still find the manifest via the bounded walk")
		}
	})

	t.Run("manifest must be a file", func(t *testing.T) {
		root := t.TempDir()
		// plugin.json as a DIRECTORY should not count.
		if err := os.MkdirAll(
			filepath.Join(root, ".claude-plugin", "plugin.json"), 0o755); err != nil {
			t.Fatal(err)
		}
		agentPath := filepath.Join(root, "agents", "a.md")
		if err := os.MkdirAll(filepath.Dir(agentPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(agentPath, []byte("---\nname: a\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var a Agent
		MarkPluginDistributed(&a, agentPath, root)
		if a.PluginDistributed {
			t.Error("plugin.json directory should not mark the agent")
		}
	})
}
