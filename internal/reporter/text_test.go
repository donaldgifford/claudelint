package reporter

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/diag"
)

func TestTextEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := Text(&buf, Summary{Files: 3}); err != nil {
		t.Fatalf("Text = %v, want nil", err)
	}
	want := "0 diagnostics, 3 files checked\n"
	if got := buf.String(); got != want {
		t.Errorf("Text empty = %q, want %q", got, want)
	}
}

func TestTextPositioned(t *testing.T) {
	var buf bytes.Buffer
	s := Summary{
		Files: 1,
		Diagnostics: []diag.Diagnostic{{
			RuleID:   "skills/require-name",
			Severity: diag.SeverityError,
			Path:     "skills/a/SKILL.md",
			Range: diag.Range{
				Start: diag.Position{Line: 2, Column: 1, Offset: 5},
				End:   diag.Position{Line: 2, Column: 10, Offset: 14},
			},
			Message: `frontmatter missing "name"`,
		}},
	}
	if err := Text(&buf, s); err != nil {
		t.Fatalf("Text = %v, want nil", err)
	}
	want := "skills/a/SKILL.md:2:1: error: frontmatter missing \"name\" [skills/require-name]\n" +
		"1 diagnostics, 1 files checked\n"
	if got := buf.String(); got != want {
		t.Errorf("Text positioned =\n  %q\nwant\n  %q", got, want)
	}
}

func TestTextWithColor(t *testing.T) {
	var buf bytes.Buffer
	s := Summary{
		Files: 1,
		Diagnostics: []diag.Diagnostic{{
			RuleID:   "a/x",
			Severity: diag.SeverityError,
			Path:     "x.md",
			Range:    diag.Range{Start: diag.Position{Line: 1, Column: 1}},
			Message:  "boom",
		}},
	}
	if err := TextWithOptions(&buf, s, TextOptions{Color: true}); err != nil {
		t.Fatalf("TextWithOptions = %v, want nil", err)
	}
	got := buf.String()
	// Red ANSI prefix + reset must wrap the severity.
	if !strings.Contains(got, "\x1b[31merror\x1b[0m") {
		t.Errorf("color=true output missing red severity wrap:\n%s", got)
	}
}

func TestTextWithoutColor(t *testing.T) {
	var buf bytes.Buffer
	s := Summary{
		Files: 1,
		Diagnostics: []diag.Diagnostic{{
			RuleID:   "a/x",
			Severity: diag.SeverityError,
			Path:     "x.md",
			Range:    diag.Range{Start: diag.Position{Line: 1, Column: 1}},
			Message:  "boom",
		}},
	}
	if err := TextWithOptions(&buf, s, TextOptions{Color: false}); err != nil {
		t.Fatalf("TextWithOptions = %v, want nil", err)
	}
	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("color=false output contained ANSI escape: %q", buf.String())
	}
}

func TestShouldUseColor(t *testing.T) {
	// --no-color wins over everything.
	t.Setenv("NO_COLOR", "")
	if got := ShouldUseColor(true, true); got {
		t.Errorf("--no-color should suppress color even when intent=true")
	}
	// NO_COLOR env wins over intent.
	t.Setenv("NO_COLOR", "1")
	if got := ShouldUseColor(false, true); got {
		t.Errorf("NO_COLOR env should suppress color even when intent=true")
	}
	// Neither flag nor env: intent controls.
	if err := os.Unsetenv("NO_COLOR"); err != nil {
		t.Fatalf("unset NO_COLOR: %v", err)
	}
	if got := ShouldUseColor(false, true); !got {
		t.Errorf("no inhibitors + intent=true should enable color")
	}
	if got := ShouldUseColor(false, false); got {
		t.Errorf("no intent should leave color off")
	}
}

func TestTextFileLevelDiagnosticOmitsPosition(t *testing.T) {
	var buf bytes.Buffer
	s := Summary{
		Files: 1,
		Diagnostics: []diag.Diagnostic{{
			RuleID:   "schema/parse",
			Severity: diag.SeverityError,
			Path:     ".claude/hooks/bad.json",
			Message:  "invalid JSON",
		}},
	}
	if err := Text(&buf, s); err != nil {
		t.Fatalf("Text = %v, want nil", err)
	}
	want := ".claude/hooks/bad.json: error: invalid JSON [schema/parse]\n" +
		"1 diagnostics, 1 files checked\n"
	if got := buf.String(); got != want {
		t.Errorf("Text file-level =\n  %q\nwant\n  %q", got, want)
	}
}
