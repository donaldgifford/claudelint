package claudemd

import (
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
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
