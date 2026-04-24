package discovery

import (
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantOK  bool
		wantKnd artifact.ArtifactKind
	}{
		{"root CLAUDE.md", "CLAUDE.md", true, artifact.KindClaudeMD},
		{"nested CLAUDE.md", "sub/pkg/CLAUDE.md", true, artifact.KindClaudeMD},
		{"plugin.json at root", "plugin.json", true, artifact.KindPlugin},
		{"plugin.yaml nested", "tools/myplugin/plugin.yaml", true, artifact.KindPlugin},
		{"plugin.yml nested", "tools/myplugin/plugin.yml", true, artifact.KindPlugin},
		{"settings.json", ".claude/settings.json", true, artifact.KindHook},
		{"settings.local.json", ".claude/settings.local.json", true, artifact.KindHook},
		{"hooks dir json", ".claude/hooks/precommit.json", true, artifact.KindHook},
		{"slash command", ".claude/commands/review.md", true, artifact.KindCommand},
		{"agent", ".claude/agents/refactorer.md", true, artifact.KindAgent},
		{"skill", ".claude/skills/writer/SKILL.md", true, artifact.KindSkill},
		{"plugin-embedded skill", "plugin/a/.claude/skills/x/SKILL.md", true, artifact.KindSkill},

		// Plugin-distribution layouts (no .claude/ parent).
		{"plugin-root skill", "skills/go/SKILL.md", true, artifact.KindSkill},
		{"plugin-root command", "commands/review.md", true, artifact.KindCommand},
		{"plugin-root agent", "agents/refactor.md", true, artifact.KindAgent},
		{"plugin-root hook", "hooks/pre.json", true, artifact.KindHook},
		{"plugin-versioned skill", "go-development/2.0.1/skills/go/SKILL.md", true, artifact.KindSkill},
		{"plugin-versioned command", "go-development/2.0.1/commands/review.md", true, artifact.KindCommand},

		// Marketplace manifest at repo root and nested.
		{"marketplace root", ".claude-plugin/marketplace.json", true, artifact.KindMarketplace},
		{"marketplace nested", "some/sub/.claude-plugin/marketplace.json", true, artifact.KindMarketplace},
		{"marketplace requires dir prefix", "marketplace.json", false, ""},

		// MCP server declarations.
		{"mcp at root", ".mcp.json", true, artifact.KindMCPServer},
		{"mcp nested", "plugins/foo/.mcp.json", true, artifact.KindMCPServer},

		{
			"skill companion file is not a skill artifact",
			".claude/skills/writer/references/style.md", false, "",
		},
		{
			"non-SKILL md in skill dir",
			".claude/skills/writer/notes.md", false, "",
		},
		{"hooks dir non-json", ".claude/hooks/readme.md", false, ""},
		{"command dir non-md", ".claude/commands/review.txt", false, ""},
		{"unrelated file", "src/main.go", false, ""},
		{"empty path", "", false, ""},
		{"dot path", ".", false, ""},
		{"absolute path rejected", "/abs/CLAUDE.md", false, ""},
		{"parent escape rejected", "../CLAUDE.md", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKnd, gotOK := Classify(tt.path)
			if gotOK != tt.wantOK {
				t.Fatalf("Classify(%q) ok = %v, want %v", tt.path, gotOK, tt.wantOK)
			}
			if gotKnd != tt.wantKnd {
				t.Errorf("Classify(%q) kind = %q, want %q", tt.path, gotKnd, tt.wantKnd)
			}
		})
	}
}

// TestClassifyCoversEveryKind guards that the classification table in
// this file keeps up with artifact.AllKinds — a new ArtifactKind has
// to get at least one positive case or the test fails.
func TestClassifyCoversEveryKind(t *testing.T) {
	fixtures := map[artifact.ArtifactKind]string{
		artifact.KindClaudeMD:    "CLAUDE.md",
		artifact.KindSkill:       ".claude/skills/x/SKILL.md",
		artifact.KindCommand:     ".claude/commands/x.md",
		artifact.KindAgent:       ".claude/agents/x.md",
		artifact.KindHook:        ".claude/settings.json",
		artifact.KindPlugin:      "plugin.json",
		artifact.KindMarketplace: ".claude-plugin/marketplace.json",
		artifact.KindMCPServer:   ".mcp.json",
	}
	for _, k := range artifact.AllKinds() {
		p, ok := fixtures[k]
		if !ok {
			t.Errorf("no classify fixture for kind %q — add one", k)
			continue
		}
		got, ok := Classify(p)
		if !ok || got != k {
			t.Errorf("Classify(%q) = (%q, %v), want (%q, true)", p, got, ok, k)
		}
	}
}
