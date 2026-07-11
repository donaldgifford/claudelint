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

func TestNoVersionFieldPresent(t *testing.T) {
	src := []byte("---\nname: x\ndescription: y\nversion: 1.2.3\n---\n# body\n")
	s, _ := artifact.ParseSkill("s.md", src)
	d := (&noVersionField{}).Check(&optCtx{}, s)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic for SKILL.md with version, got %d", len(d))
	}
	if !strings.Contains(d[0].Message, "plugin.json") {
		t.Errorf("message should mention plugin.json as canonical source, got %q", d[0].Message)
	}
	// Range must point at the version key (line 4 of the source —
	// dashes line 1, name line 2, description line 3, version line 4).
	if d[0].Range.Start.Line != 4 {
		t.Errorf("Range.Start.Line = %d, want 4", d[0].Range.Start.Line)
	}
	if d[0].Range.IsZero() {
		t.Errorf("Range is zero; should point at the version key")
	}
}

func TestNoVersionFieldAbsent(t *testing.T) {
	src := []byte("---\nname: x\ndescription: y\n---\n# body\n")
	s, _ := artifact.ParseSkill("s.md", src)
	if d := (&noVersionField{}).Check(&optCtx{}, s); len(d) != 0 {
		t.Errorf("SKILL.md without version should pass, got %v", d)
	}
}

// TestNoVersionFieldCommentedOut covers the success criterion that a
// commented-out `# version:` line in the body (or anywhere outside
// the YAML frontmatter) does not trigger the rule. The Skill parser
// only parses keys inside the --- frontmatter fence, so a `version`
// elsewhere in the document is invisible to Frontmatter.Keys.
func TestNoVersionFieldCommentedOut(t *testing.T) {
	src := []byte("---\nname: x\ndescription: y\n---\n# body\n# version: 1.0\n")
	s, _ := artifact.ParseSkill("s.md", src)
	if d := (&noVersionField{}).Check(&optCtx{}, s); len(d) != 0 {
		t.Errorf("commented-out version in body should not trigger, got %v", d)
	}
}

func TestDescriptionLength(t *testing.T) {
	long := strings.Repeat("x", 1600)
	half := strings.Repeat("y", 800)
	cases := []struct {
		name  string
		fm    string
		wantN int
	}{
		{"short description", "description: does the thing\n", 0},
		{"long description alone", "description: " + long + "\n", 1},
		{"combined overflow", "description: " + half + "\nwhen_to_use: " + half + "\n", 1},
		{"combined under limit", "description: " + half[:400] + "\nwhen_to_use: " + half[:400] + "\n", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte("---\nname: s\n" + tc.fm + "---\nbody\n")
			s, perr := artifact.ParseSkill("skills/s/SKILL.md", src)
			if perr != nil {
				t.Fatalf("ParseSkill = %v", perr)
			}
			d := (&descriptionLength{}).Check(&optCtx{}, s)
			if len(d) != tc.wantN {
				t.Fatalf("got %d diagnostics, want %d", len(d), tc.wantN)
			}
			if tc.wantN == 1 {
				if !strings.Contains(d[0].Message, "limit 1536") {
					t.Errorf("message = %q", d[0].Message)
				}
				if d[0].Range.IsZero() {
					t.Errorf("diagnostic should anchor at the description key")
				}
			}
		})
	}

	// Option override tightens the budget.
	src := []byte("---\nname: s\ndescription: " + strings.Repeat("z", 200) + "\n---\nbody\n")
	s, _ := artifact.ParseSkill("skills/s/SKILL.md", src)
	strict := &optCtx{opts: map[string]any{"max_chars": 100}}
	if d := (&descriptionLength{}).Check(strict, s); len(d) != 1 {
		t.Errorf("max_chars=100 should flag a 200-char description, got %v", d)
	}
}

func TestForkAgentPairing(t *testing.T) {
	cases := []struct {
		name  string
		fm    string
		wantN int
	}{
		{"no agent", "", 0},
		{"agent with fork", "context: fork\nagent: shipper\n", 0},
		{"agent without context", "agent: shipper\n", 1},
		{"agent with non-fork context", "context: session\nagent: shipper\n", 1},
		{"fork without agent is fine", "context: fork\n", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte("---\nname: s\ndescription: d\n" + tc.fm + "---\nbody\n")
			s, perr := artifact.ParseSkill("skills/s/SKILL.md", src)
			if perr != nil {
				t.Fatalf("ParseSkill = %v", perr)
			}
			d := (&forkAgentPairing{}).Check(nil, s)
			if len(d) != tc.wantN {
				t.Fatalf("got %d diagnostics, want %d (%v)", len(d), tc.wantN, d)
			}
			if tc.wantN == 1 {
				if !strings.Contains(d[0].Message, "context: fork") {
					t.Errorf("message = %q", d[0].Message)
				}
				if d[0].Range.IsZero() {
					t.Errorf("diagnostic should anchor at the agent key")
				}
			}
		})
	}
}
