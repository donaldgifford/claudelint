package agents

import (
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// newAgent parses a minimal agent file with the given extra
// frontmatter lines so rule tests exercise the real parser path.
func newAgent(t *testing.T, extra string) *artifact.Agent {
	t.Helper()
	src := []byte("---\nname: helper\ndescription: d\n" + extra + "---\nbody\n")
	a, perr := artifact.ParseAgent(".claude/agents/helper.md", src)
	if perr != nil {
		t.Fatalf("ParseAgent = %v", perr)
	}
	return a
}

func TestModelValid(t *testing.T) {
	cases := []struct {
		name  string
		model string // empty = omit the key
		wantN int
	}{
		{"omitted means inherit", "", 0},
		{"alias sonnet", "sonnet", 0},
		{"alias fable", "fable", 0},
		{"inherit", "inherit", 0},
		{"full model ID", "claude-opus-4-8", 0},
		{"typo sonet", "sonet", 1},
		{"uppercase alias", "Sonnet", 1},
		{"bogus", "gpt-4", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			extra := ""
			if tc.model != "" {
				extra = "model: " + tc.model + "\n"
			}
			a := newAgent(t, extra)
			d := (&modelValid{}).Check(nil, a)
			if len(d) != tc.wantN {
				t.Fatalf("got %d diagnostics, want %d (%v)", len(d), tc.wantN, d)
			}
			if tc.wantN == 1 {
				if !strings.Contains(d[0].Message, "sonnet, opus, haiku, fable, inherit") {
					t.Errorf("message should name the valid values, got %q", d[0].Message)
				}
				if d[0].Range.IsZero() {
					t.Errorf("diagnostic should anchor at the model key range")
				}
			}
		})
	}
}

func TestNameFormat(t *testing.T) {
	cases := []struct {
		name      string
		agentName string
		wantN     int
	}{
		{"simple", "scribe", 0},
		{"hyphenated", "code-reviewer", 0},
		{"uppercase", "Reviewer", 1},
		{"underscore", "code_reviewer", 1},
		{"digits", "agent2", 1},
		{"trailing hyphen", "reviewer-", 1},
		{"double hyphen", "code--reviewer", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte("---\nname: " + tc.agentName + "\ndescription: d\n---\nbody\n")
			a, perr := artifact.ParseAgent(".claude/agents/x.md", src)
			if perr != nil {
				t.Fatalf("ParseAgent = %v", perr)
			}
			d := (&nameFormat{}).Check(nil, a)
			if len(d) != tc.wantN {
				t.Errorf("got %d diagnostics, want %d (%v)", len(d), tc.wantN, d)
			}
		})
	}

	// Empty name is frontmatter-required's finding, not ours.
	empty := &artifact.Agent{}
	if d := (&nameFormat{}).Check(nil, empty); len(d) != 0 {
		t.Errorf("empty name should be skipped, got %v", d)
	}
}

func TestToolsKnown(t *testing.T) {
	cases := []struct {
		name  string
		extra string
		wantN int
	}{
		{"no lists", "", 0},
		{"valid names and patterns", "tools: Read, Grep, mcp__github, Bash(git diff:*)\n", 0},
		{"typo in tools", "tools: Read, Wrte\n", 1},
		{"typo in disallowedTools", "disallowedTools: Wrte\n", 1},
		{"typos in both", "tools: Frob\ndisallowedTools: Nicate\n", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newAgent(t, tc.extra)
			d := (&toolsKnown{}).Check(nil, a)
			if len(d) != tc.wantN {
				t.Fatalf("got %d diagnostics, want %d (%v)", len(d), tc.wantN, d)
			}
			for _, dd := range d {
				if !strings.Contains(dd.Message, "silently ignores") {
					t.Errorf("message should explain the silent-ignore behavior, got %q", dd.Message)
				}
				if dd.Range.IsZero() {
					t.Errorf("diagnostic should anchor at the offending key range")
				}
			}
		})
	}
}

func TestModelValidRunsOnSkillsAndCommands(t *testing.T) {
	s, perr := artifact.ParseSkill("skills/x/SKILL.md",
		[]byte("---\nname: x\ndescription: d\nmodel: sonet\n---\nbody\n"))
	if perr != nil {
		t.Fatalf("ParseSkill = %v", perr)
	}
	if d := (&modelValid{}).Check(nil, s); len(d) != 1 {
		t.Errorf("skill with typo'd model: want 1, got %d", len(d))
	}

	c, perr := artifact.ParseCommand(".claude/commands/x.md",
		[]byte("---\ndescription: d\nmodel: haiku\n---\nbody\n"))
	if perr != nil {
		t.Fatalf("ParseCommand = %v", perr)
	}
	if d := (&modelValid{}).Check(nil, c); len(d) != 0 {
		t.Errorf("command with valid model: want 0, got %v", d)
	}
}
