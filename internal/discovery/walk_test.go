package discovery

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// fixtureTree writes a minimal repo under t.TempDir containing one
// example per ArtifactKind plus a few negative cases. It returns the
// root path. HOME and XDG_CONFIG_HOME are redirected to the temp dir
// so the user-global gitignore cannot leak into tests.
func fixtureTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	isolated := t.TempDir()
	t.Setenv("HOME", isolated)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(isolated, ".config"))

	files := map[string]string{
		"CLAUDE.md":                             "# root claude",
		"subdir/CLAUDE.md":                      "# nested claude",
		"plugin.json":                           `{"name":"p"}`,
		".claude/settings.json":                 `{"hooks":{}}`,
		".claude/settings.local.json":           `{}`,
		".claude/hooks/precommit.json":          `{"event":"PreToolUse"}`,
		".claude/commands/review.md":            "---\nname: r\n---\n",
		".claude/agents/refactorer.md":          "---\nname: a\n---\n",
		".claude/skills/writer/SKILL.md":        "---\nname: w\n---\n",
		".claude/skills/writer/references/x.md": "companion, not an artifact",
		"src/main.go":                           "// ignored",
		"README.md":                             "# readme (ignored)",
	}

	for rel, body := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return root
}

func TestWalkDiscoversEveryKind(t *testing.T) {
	root := fixtureTree(t)

	w := New(Options{})
	got, err := w.Walk(root)
	if err != nil {
		t.Fatalf("Walk(%s) = %v, want nil", root, err)
	}

	wantByPath := map[string]artifact.ArtifactKind{
		"CLAUDE.md":                      artifact.KindClaudeMD,
		"plugin.json":                    artifact.KindPlugin,
		"subdir/CLAUDE.md":               artifact.KindClaudeMD,
		".claude/settings.json":          artifact.KindHook,
		".claude/settings.local.json":    artifact.KindHook,
		".claude/hooks/precommit.json":   artifact.KindHook,
		".claude/commands/review.md":     artifact.KindCommand,
		".claude/agents/refactorer.md":   artifact.KindAgent,
		".claude/skills/writer/SKILL.md": artifact.KindSkill,
	}

	gotByPath := make(map[string]artifact.ArtifactKind, len(got))
	for _, c := range got {
		gotByPath[c.Path] = c.Kind
	}

	for p, want := range wantByPath {
		g, ok := gotByPath[p]
		if !ok {
			t.Errorf("Walk() missing %q", p)
			continue
		}
		if g != want {
			t.Errorf("Walk()[%q] = %q, want %q", p, g, want)
		}
	}

	// Negative: src/main.go, README.md, and the skill companion should
	// not appear.
	for _, p := range []string{
		"src/main.go",
		"README.md",
		".claude/skills/writer/references/x.md",
	} {
		if _, ok := gotByPath[p]; ok {
			t.Errorf("Walk() should not have returned %q", p)
		}
	}
}

func TestWalkHonorsRootGitignore(t *testing.T) {
	root := fixtureTree(t)

	// Root .gitignore excludes plugin.json.
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("plugin.json\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	got, err := New(Options{}).Walk(root)
	if err != nil {
		t.Fatalf("Walk = %v", err)
	}
	for _, c := range got {
		if c.Path == "plugin.json" {
			t.Errorf("gitignored plugin.json still returned")
		}
	}
}

func TestWalkHonorsNestedGitignore(t *testing.T) {
	root := fixtureTree(t)

	// subdir/.gitignore excludes CLAUDE.md only within subdir.
	gi := filepath.Join(root, "subdir", ".gitignore")
	if err := os.WriteFile(gi, []byte("CLAUDE.md\n"), 0o644); err != nil {
		t.Fatalf("write nested .gitignore: %v", err)
	}

	got, err := New(Options{}).Walk(root)
	if err != nil {
		t.Fatalf("Walk = %v", err)
	}
	paths := make([]string, 0, len(got))
	for _, c := range got {
		paths = append(paths, c.Path)
	}

	if slices.Contains(paths, "subdir/CLAUDE.md") {
		t.Errorf("nested-ignored subdir/CLAUDE.md still returned; got %v", paths)
	}
	if !slices.Contains(paths, "CLAUDE.md") {
		t.Errorf("root CLAUDE.md was dropped by nested .gitignore — scoping broken")
	}
}

func TestWalkHonorsExtraIgnore(t *testing.T) {
	root := fixtureTree(t)

	got, err := New(Options{ExtraIgnore: []string{"**/commands/*.md"}}).Walk(root)
	if err != nil {
		t.Fatalf("Walk = %v", err)
	}
	for _, c := range got {
		if c.Kind == artifact.KindCommand {
			t.Errorf("ExtraIgnore glob did not suppress command %q", c.Path)
		}
	}
}

func TestWalkStableOrder(t *testing.T) {
	root := fixtureTree(t)

	first, err := New(Options{}).Walk(root)
	if err != nil {
		t.Fatalf("Walk = %v", err)
	}
	second, err := New(Options{}).Walk(root)
	if err != nil {
		t.Fatalf("Walk = %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("Walk returned %d then %d candidates", len(first), len(second))
	}
	for i := range first {
		if first[i].Path != second[i].Path {
			t.Errorf("order drifted at %d: %q vs %q", i, first[i].Path, second[i].Path)
		}
	}
}

func TestWalkHonorsGitInfoExclude(t *testing.T) {
	root := fixtureTree(t)

	if err := os.MkdirAll(filepath.Join(root, ".git", "info"), 0o755); err != nil {
		t.Fatalf("mkdir .git/info: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, ".git", "info", "exclude"),
		[]byte(".claude/commands/review.md\n"),
		0o644,
	); err != nil {
		t.Fatalf("write exclude: %v", err)
	}

	got, err := New(Options{}).Walk(root)
	if err != nil {
		t.Fatalf("Walk = %v", err)
	}
	for _, c := range got {
		if c.Path == ".claude/commands/review.md" {
			t.Errorf("git-info-exclude did not suppress %q", c.Path)
		}
	}
}

func TestWalkHonorsGlobalIgnore(t *testing.T) {
	root := fixtureTree(t)

	global := filepath.Join(t.TempDir(), "global_ignore")
	if err := os.WriteFile(global, []byte(".claude/agents/*.md\n"), 0o644); err != nil {
		t.Fatalf("write global: %v", err)
	}

	got, err := New(Options{GlobalIgnorePath: global}).Walk(root)
	if err != nil {
		t.Fatalf("Walk = %v", err)
	}
	for _, c := range got {
		if c.Kind == artifact.KindAgent {
			t.Errorf("global ignore did not suppress agent %q", c.Path)
		}
	}
}

func TestWalkSingleFile(t *testing.T) {
	root := fixtureTree(t)

	p := filepath.Join(root, "CLAUDE.md")
	got, err := New(Options{}).Walk(p)
	if err != nil {
		t.Fatalf("Walk(file) = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Walk(file) = %d candidates, want 1", len(got))
	}
	if got[0].Kind != artifact.KindClaudeMD {
		t.Errorf("Walk(file) kind = %q, want %q", got[0].Kind, artifact.KindClaudeMD)
	}
}
