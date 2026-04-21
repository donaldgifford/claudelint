package skills

import (
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// optCtx is a test-only rules.Context backed by an in-memory map.
type optCtx struct{ opts map[string]any }

func (*optCtx) RuleID() string          { return "" }
func (c *optCtx) Option(key string) any { return c.opts[key] }
func (*optCtx) Logf(_ string, _ ...any) {}

func TestBodySizeUnder(t *testing.T) {
	src := []byte("---\nname: x\ndescription: y\n---\nhello world\n")
	s, _ := artifact.ParseSkill("s.md", src)
	r := &bodySize{}
	d := r.Check(&optCtx{opts: map[string]any{"max_words": 100}}, s)
	if len(d) != 0 {
		t.Errorf("short body should emit no diagnostics, got %v", d)
	}
}

func TestBodySizeOver(t *testing.T) {
	body := strings.Repeat("word ", 1500)
	src := []byte("---\nname: x\ndescription: y\n---\n" + body)
	s, _ := artifact.ParseSkill("s.md", src)
	r := &bodySize{}
	d := r.Check(&optCtx{opts: map[string]any{"max_words": 1000}}, s)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
	if !strings.Contains(d[0].Message, "1500 words") {
		t.Errorf("message = %q", d[0].Message)
	}
}

func TestTriggerClarityPresent(t *testing.T) {
	src := []byte("---\nname: x\ndescription: Use when drafting emails\n---\n")
	s, _ := artifact.ParseSkill("s.md", src)
	r := &triggerClarity{}
	d := r.Check(&optCtx{}, s)
	if len(d) != 0 {
		t.Errorf("trigger phrase present should pass, got %v", d)
	}
}

func TestTriggerClarityAbsent(t *testing.T) {
	src := []byte("---\nname: x\ndescription: writes emails\n---\n")
	s, _ := artifact.ParseSkill("s.md", src)
	r := &triggerClarity{}
	d := r.Check(&optCtx{}, s)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %+v", d)
	}
}

func TestTriggerClaritySkipsWhenDescriptionEmpty(t *testing.T) {
	src := []byte("---\nname: x\n---\n")
	s, _ := artifact.ParseSkill("s.md", src)
	r := &triggerClarity{}
	d := r.Check(&optCtx{}, s)
	if len(d) != 0 {
		t.Errorf("empty description should not trigger trigger-clarity (frontmatter-required owns that)")
	}
}
