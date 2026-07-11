package commands

import (
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

func TestAllowedToolsKnownOK(t *testing.T) {
	src := []byte("---\ndescription: x\nallowed-tools:\n  - Read\n  - Bash\n---\n")
	c, _ := artifact.ParseCommand("c.md", src)
	r := &allowedToolsKnown{}
	if d := r.Check(nil, c); len(d) != 0 {
		t.Errorf("expected no diagnostics, got %v", d)
	}
}

func TestAllowedToolsKnownRejectsUnknown(t *testing.T) {
	src := []byte("---\ndescription: x\nallowed-tools:\n  - Read\n  - Typo\n  - Bash\n---\n")
	c, _ := artifact.ParseCommand("c.md", src)
	r := &allowedToolsKnown{}
	d := r.Check(nil, c)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
}

func TestAllowedToolsKnownEmptyList(t *testing.T) {
	src := []byte("---\ndescription: x\n---\n")
	c, _ := artifact.ParseCommand("c.md", src)
	r := &allowedToolsKnown{}
	if d := r.Check(nil, c); len(d) != 0 {
		t.Errorf("absent allowed-tools should emit nothing, got %v", d)
	}
}

func TestAllowedToolsKnownAcceptsPatterns(t *testing.T) {
	src := []byte("---\ndescription: x\n" +
		"allowed-tools: Bash(git add:*), mcp__github, mcp__linear__create_issue, Agent(reviewer)\n---\n")
	c, _ := artifact.ParseCommand("c.md", src)
	r := &allowedToolsKnown{}
	if d := r.Check(nil, c); len(d) != 0 {
		t.Errorf("permission rules and mcp patterns should pass, got %v", d)
	}
}

func TestAllowedToolsKnownChecksDisallowedTools(t *testing.T) {
	src := []byte("---\ndescription: x\ndisallowed-tools:\n  - WriteFil\n---\n")
	c, _ := artifact.ParseCommand("c.md", src)
	r := &allowedToolsKnown{}
	d := r.Check(nil, c)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
	if !strings.Contains(d[0].Message, "disallowed-tools") {
		t.Errorf("message should name the disallowed-tools key, got %q", d[0].Message)
	}
}

func TestAllowedToolsKnownRunsOnSkills(t *testing.T) {
	src := []byte("---\nname: s\ndescription: x\n" +
		"allowed-tools: Read, Typo\ndisallowed-tools: WriteFil\n---\nbody\n")
	s, _ := artifact.ParseSkill("skills/s/SKILL.md", src)
	r := &allowedToolsKnown{}
	d := r.Check(nil, s)
	if len(d) != 2 {
		t.Fatalf("expected 2 diagnostics (Typo + WriteFil), got %d: %v", len(d), d)
	}

	ok := []byte("---\nname: s\ndescription: x\nallowed-tools: Read, Bash(just test:*)\n---\nbody\n")
	s2, _ := artifact.ParseSkill("skills/s/SKILL.md", ok)
	if d := r.Check(nil, s2); len(d) != 0 {
		t.Errorf("valid skill tool list should pass, got %v", d)
	}
}
