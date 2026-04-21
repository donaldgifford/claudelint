package reporter

import (
	"bytes"
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
