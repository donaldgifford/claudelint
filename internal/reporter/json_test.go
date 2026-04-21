package reporter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// TestJSONEmpty locks in the shape of a zero-diagnostic report. The
// expected bytes are the contract: every key is present, counts are
// zero, diagnostics is [] (never null).
func TestJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, Summary{Files: 3}); err != nil {
		t.Fatalf("JSON = %v, want nil", err)
	}

	var got jsonReport
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, buf.String())
	}
	if got.SchemaVersion != JSONSchemaVersion {
		t.Errorf("schema_version = %q, want %q", got.SchemaVersion, JSONSchemaVersion)
	}
	if got.RulesetVersion != rules.RulesetVersion {
		t.Errorf("ruleset_version = %q, want %q", got.RulesetVersion, rules.RulesetVersion)
	}
	if got.FilesChecked != 3 {
		t.Errorf("files_checked = %d, want 3", got.FilesChecked)
	}
	if got.DiagnosticCount != 0 {
		t.Errorf("diagnostic_count = %d, want 0", got.DiagnosticCount)
	}
	if got.SeverityCount != (jsonSeverityCount{}) {
		t.Errorf("severity_count = %+v, want all zero", got.SeverityCount)
	}
	// diagnostics must encode as [], not null.
	if !strings.Contains(buf.String(), `"diagnostics": []`) {
		t.Errorf("empty diagnostics should encode as [], got:\n%s", buf.String())
	}
}

func TestJSONWithDiagnostics(t *testing.T) {
	var buf bytes.Buffer
	s := Summary{
		Files: 2,
		Diagnostics: []diag.Diagnostic{
			{
				RuleID:   "skills/require-name",
				Severity: diag.SeverityError,
				Path:     "skills/a/SKILL.md",
				Range: diag.Range{
					Start: diag.Position{Line: 2, Column: 1, Offset: 5},
					End:   diag.Position{Line: 2, Column: 10, Offset: 14},
				},
				Message: "frontmatter missing \"name\"",
			},
			{
				RuleID:   "style/no-emoji",
				Severity: diag.SeverityWarning,
				Path:     "CLAUDE.md",
				Range:    diag.Range{Start: diag.Position{Line: 5, Column: 3}},
				Message:  "avoid emoji",
			},
		},
	}
	if err := JSON(&buf, s); err != nil {
		t.Fatalf("JSON = %v, want nil", err)
	}

	var got jsonReport
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, buf.String())
	}
	if got.DiagnosticCount != 2 {
		t.Errorf("diagnostic_count = %d, want 2", got.DiagnosticCount)
	}
	if got.SeverityCount.Error != 1 || got.SeverityCount.Warning != 1 {
		t.Errorf("severity_count = %+v, want error=1 warning=1", got.SeverityCount)
	}
	// Severity encodes as string.
	raw := buf.String()
	if !strings.Contains(raw, `"severity": "error"`) {
		t.Errorf("severity string form missing from output:\n%s", raw)
	}
	if !strings.Contains(raw, `"severity": "warning"`) {
		t.Errorf("severity warning missing from output:\n%s", raw)
	}
	// Fix must be omitted entirely when nil (forward compat).
	if strings.Contains(raw, `"fix"`) {
		t.Errorf("fix field should be omitted when nil, got:\n%s", raw)
	}
}

// TestJSONFingerprintStable guards that every run of a given claudelint
// binary emits the same fingerprint. The fingerprint is a function of
// the registered ruleset alone; if this changes mid-run something has
// corrupted package state.
func TestJSONFingerprintStable(t *testing.T) {
	var a, b bytes.Buffer
	_ = JSON(&a, Summary{})
	_ = JSON(&b, Summary{})
	if a.String() != b.String() {
		t.Errorf("JSON output differs across calls:\n%s\nvs\n%s", a.String(), b.String())
	}
}
