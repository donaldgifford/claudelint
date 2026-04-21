package commands

import (
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
