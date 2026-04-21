package artifact

import (
	"errors"
	"testing"

	"github.com/donaldgifford/claudelint/internal/diag"
)

func TestParseErrorErrorFileLevel(t *testing.T) {
	e := &ParseError{Path: "skills/a/SKILL.md", Message: "empty file"}
	want := "skills/a/SKILL.md: empty file"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestParseErrorErrorPositioned(t *testing.T) {
	e := &ParseError{
		Path:    "plugins/x/plugin.json",
		Range:   diag.Range{Start: diag.Position{Line: 7, Column: 12}},
		Message: "invalid JSON",
	}
	want := "plugins/x/plugin.json:7:12: invalid JSON"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestParseErrorUnwrap(t *testing.T) {
	inner := errors.New("library boom")
	e := &ParseError{Path: "x", Message: "m", Cause: inner}
	if !errors.Is(e, inner) {
		t.Errorf("errors.Is(ParseError, inner) = false, want true")
	}
}
