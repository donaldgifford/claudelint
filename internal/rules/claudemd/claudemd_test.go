package claudemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
)

type optCtx struct{ opts map[string]any }

func (*optCtx) RuleID() string          { return "" }
func (c *optCtx) Option(k string) any   { return c.opts[k] }
func (*optCtx) Logf(_ string, _ ...any) {}

func TestSizeUnder(t *testing.T) {
	src := []byte("a\n")
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	if d := (&size{}).Check(&optCtx{opts: map[string]any{"max_lines": 100}}, c); len(d) != 0 {
		t.Errorf("expected no diagnostics, got %v", d)
	}
}

func TestSizeOver(t *testing.T) {
	src := []byte(strings.Repeat("line\n", 600))
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	d := (&size{}).Check(&optCtx{opts: map[string]any{"max_lines": 500}}, c)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
}

func TestDuplicateDirectives(t *testing.T) {
	src := []byte("# Rules\n\n- Use tests.\n- Write docs.\n- use tests\n")
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	d := (&duplicateDirectives{}).Check(nil, c)
	if len(d) != 1 {
		t.Fatalf("expected 1 duplicate, got %d", len(d))
	}
	if !strings.Contains(d[0].Message, "line 3") {
		t.Errorf("message should cite the first occurrence, got %q", d[0].Message)
	}
}

func TestDuplicateDirectivesIgnoresNonBullets(t *testing.T) {
	src := []byte("plain paragraph.\nanother plain paragraph.\n")
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	if d := (&duplicateDirectives{}).Check(nil, c); len(d) != 0 {
		t.Errorf("plain paragraphs should not register as duplicates, got %v", d)
	}
}

// writeMD writes content to name under dir and returns the full path.
func writeMD(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// checkImports parses a CLAUDE.md at dir with the given body and runs
// importExists over it.
func checkImports(t *testing.T, dir, body string) []diag.Diagnostic {
	t.Helper()
	p := writeMD(t, dir, "CLAUDE.md", body)
	c, perr := artifact.ParseClaudeMD(p, []byte(body))
	if perr != nil {
		t.Fatalf("ParseClaudeMD = %v", perr)
	}
	return (&importExists{}).Check(nil, c)
}

func TestImportExists(t *testing.T) {
	t.Run("resolving import is silent", func(t *testing.T) {
		dir := t.TempDir()
		writeMD(t, dir, "README.md", "hello\n")
		if d := checkImports(t, dir, "See @README.md for details.\n"); len(d) != 0 {
			t.Errorf("resolving import flagged: %v", d)
		}
	})

	t.Run("missing import flags with range", func(t *testing.T) {
		dir := t.TempDir()
		d := checkImports(t, dir, "See @docs/missing.md for details.\n")
		if len(d) != 1 {
			t.Fatalf("got %d diagnostics, want 1 (%v)", len(d), d)
		}
		if !strings.Contains(d[0].Message, "@docs/missing.md does not resolve") {
			t.Errorf("message = %q", d[0].Message)
		}
		if d[0].Range.IsZero() {
			t.Errorf("diagnostic should anchor at the import token")
		}
	})

	t.Run("code span and fence are skipped", func(t *testing.T) {
		dir := t.TempDir()
		body := "Mention `@missing-span.md` literally.\n\n```text\n@missing-fence.md\n```\n"
		if d := checkImports(t, dir, body); len(d) != 0 {
			t.Errorf("code-span/fence imports should be skipped: %v", d)
		}
	})

	t.Run("email is not an import", func(t *testing.T) {
		dir := t.TempDir()
		if d := checkImports(t, dir, "Contact donald@example.com about this.\n"); len(d) != 0 {
			t.Errorf("email flagged as import: %v", d)
		}
	})

	t.Run("trailing punctuation trimmed", func(t *testing.T) {
		dir := t.TempDir()
		writeMD(t, dir, "README.md", "hi\n")
		if d := checkImports(t, dir, "Read @README.md.\n"); len(d) != 0 {
			t.Errorf("sentence-final period broke resolution: %v", d)
		}
	})

	t.Run("home import respects HOME", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		writeMD(t, home, "notes.md", "n\n")
		dir := t.TempDir()
		if d := checkImports(t, dir, "Prefs: @~/notes.md\n"); len(d) != 0 {
			t.Errorf("home import flagged: %v", d)
		}
		if d := checkImports(t, dir, "Prefs: @~/gone.md\n"); len(d) != 1 {
			t.Errorf("missing home import: got %v", d)
		}
	})

	t.Run("four hops allowed five flagged", func(t *testing.T) {
		dir := t.TempDir()
		writeMD(t, dir, "f4.md", "leaf\n")
		writeMD(t, dir, "f3.md", "@f4.md\n")
		writeMD(t, dir, "f2.md", "@f3.md\n")
		writeMD(t, dir, "f1.md", "@f2.md\n")
		if d := checkImports(t, dir, "@f1.md\n"); len(d) != 0 {
			t.Errorf("4-hop chain flagged: %v", d)
		}
		writeMD(t, dir, "f5.md", "deep leaf\n")
		writeMD(t, dir, "f4.md", "@f5.md\n")
		d := checkImports(t, dir, "@f1.md\n")
		if len(d) != 1 {
			t.Fatalf("5-hop chain: got %d diagnostics, want 1 (%v)", len(d), d)
		}
		if !strings.Contains(d[0].Message, "4-hop limit") {
			t.Errorf("message = %q", d[0].Message)
		}
	})

	t.Run("cycles terminate quietly", func(t *testing.T) {
		dir := t.TempDir()
		writeMD(t, dir, "a.md", "@b.md\n")
		writeMD(t, dir, "b.md", "@a.md\n")
		if d := checkImports(t, dir, "@a.md\n"); len(d) != 0 {
			t.Errorf("2-file cycle flagged or hung: %v", d)
		}
	})
}
