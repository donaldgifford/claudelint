package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/diag"
)

func TestGitHubEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := GitHub(&buf, Summary{Files: 2}); err != nil {
		t.Fatalf("GitHub = %v, want nil", err)
	}
	want := "0 diagnostics, 2 files checked\n"
	if got := buf.String(); got != want {
		t.Errorf("GitHub empty = %q, want %q", got, want)
	}
}

func TestGitHubSeverityCommands(t *testing.T) {
	cases := []struct {
		sev     diag.Severity
		wantCmd string
	}{
		{diag.SeverityError, "::error "},
		{diag.SeverityWarning, "::warning "},
		{diag.SeverityInfo, "::notice "},
	}
	for _, tc := range cases {
		t.Run(tc.sev.String(), func(t *testing.T) {
			var buf bytes.Buffer
			s := Summary{
				Files: 1,
				Diagnostics: []diag.Diagnostic{{
					RuleID:   "x/y",
					Severity: tc.sev,
					Path:     "a.md",
					Range:    diag.Range{Start: diag.Position{Line: 3, Column: 2}},
					Message:  "hello",
				}},
			}
			if err := GitHub(&buf, s); err != nil {
				t.Fatalf("GitHub = %v, want nil", err)
			}
			if !strings.HasPrefix(buf.String(), tc.wantCmd) {
				t.Errorf("output %q should start with %q", buf.String(), tc.wantCmd)
			}
		})
	}
}

func TestGitHubParamsFormat(t *testing.T) {
	var buf bytes.Buffer
	s := Summary{
		Files: 1,
		Diagnostics: []diag.Diagnostic{{
			RuleID:   "skills/require-name",
			Severity: diag.SeverityError,
			Path:     "skills/a/SKILL.md",
			Range:    diag.Range{Start: diag.Position{Line: 2, Column: 1}},
			Message:  `frontmatter missing "name"`,
		}},
	}
	if err := GitHub(&buf, s); err != nil {
		t.Fatalf("GitHub = %v, want nil", err)
	}
	got := buf.String()
	wantLine := "::error file=skills/a/SKILL.md,line=2,col=1,title=skills/require-name::" +
		`frontmatter missing "name"` + "\n"
	if !strings.Contains(got, wantLine) {
		t.Errorf("output does not contain expected line:\ngot:\n%s\nwant line:\n%s", got, wantLine)
	}
}

func TestGitHubOmitsPositionForFileLevel(t *testing.T) {
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
	if err := GitHub(&buf, s); err != nil {
		t.Fatalf("GitHub = %v, want nil", err)
	}
	got := buf.String()
	if strings.Contains(got, "line=") {
		t.Errorf("file-level diagnostic should omit line=: %s", got)
	}
	if !strings.Contains(got, "file=.claude/hooks/bad.json") {
		t.Errorf("file= param missing: %s", got)
	}
}

func TestGitHubEscapesSpecialChars(t *testing.T) {
	var buf bytes.Buffer
	s := Summary{
		Files: 1,
		Diagnostics: []diag.Diagnostic{{
			RuleID:   "x,y:z",
			Severity: diag.SeverityWarning,
			Path:     "weird,path:name.md",
			Range:    diag.Range{Start: diag.Position{Line: 1, Column: 1}},
			Message:  "line1\nline2 with % sign",
		}},
	}
	if err := GitHub(&buf, s); err != nil {
		t.Fatalf("GitHub = %v, want nil", err)
	}
	got := buf.String()
	// Commas in `file=` must be escaped or the param list splits wrong.
	if !strings.Contains(got, "file=weird%2Cpath%3Aname.md") {
		t.Errorf("file path commas/colons not percent-escaped: %s", got)
	}
	// `%` in message must become %25; `\n` must become %0A.
	if !strings.Contains(got, "%25") {
		t.Errorf("percent sign in message not escaped: %s", got)
	}
	if !strings.Contains(got, "%0A") {
		t.Errorf("newline in message not escaped: %s", got)
	}
}
