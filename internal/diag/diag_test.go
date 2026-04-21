package diag

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSeverityString(t *testing.T) {
	tests := []struct {
		name string
		sev  Severity
		want string
	}{
		{"info (zero value)", SeverityInfo, "info"},
		{"warning", SeverityWarning, "warning"},
		{"error", SeverityError, "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sev.String(); got != tt.want {
				t.Errorf("Severity(%d).String() = %q, want %q", tt.sev, got, tt.want)
			}
		})
	}
}

func TestSeverityRoundTrip(t *testing.T) {
	for _, sev := range []Severity{SeverityInfo, SeverityWarning, SeverityError} {
		b, err := sev.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText(%v) = %v, want nil", sev, err)
		}

		var got Severity
		if err := got.UnmarshalText(b); err != nil {
			t.Fatalf("UnmarshalText(%q) = %v, want nil", b, err)
		}
		if got != sev {
			t.Errorf("round-trip %v = %v, want %v", sev, got, sev)
		}
	}
}

func TestSeverityUnmarshalInvalid(t *testing.T) {
	var s Severity
	err := s.UnmarshalText([]byte("critical"))
	if err == nil {
		t.Fatal("UnmarshalText(\"critical\") = nil, want error")
	}
	if !strings.Contains(err.Error(), "critical") {
		t.Errorf("error message = %q, want it to contain the bad value", err.Error())
	}
}

func TestPositionIsZero(t *testing.T) {
	if got := (Position{}).IsZero(); !got {
		t.Errorf("zero Position.IsZero() = false, want true")
	}
	if got := (Position{Line: 1}).IsZero(); got {
		t.Errorf("non-zero Position.IsZero() = true, want false")
	}
}

func TestRangePointRange(t *testing.T) {
	p := Position{Line: 3, Column: 5, Offset: 42}
	r := PointRange(p)
	if r.Start != p || r.End != p {
		t.Errorf("PointRange(%+v) = %+v, want Start=End=%+v", p, r, p)
	}
}

// TestDiagnosticJSON locks the v1 JSON shape: Fix is absent when nil,
// Detail is absent when empty, and Severity is lowercased.
func TestDiagnosticJSON(t *testing.T) {
	d := Diagnostic{
		RuleID:   "skills/require-name",
		Severity: SeverityError,
		Path:     "skills/a/SKILL.md",
		Range: Range{
			Start: Position{Line: 2, Column: 1, Offset: 5},
			End:   Position{Line: 2, Column: 10, Offset: 14},
		},
		Message: `skill frontmatter is missing "name"`,
	}

	got, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("json.Marshal(Diagnostic) = %v, want nil", err)
	}
	want := `{"rule_id":"skills/require-name","severity":"error",` +
		`"path":"skills/a/SKILL.md",` +
		`"range":{"start":{"line":2,"column":1,"offset":5},` +
		`"end":{"line":2,"column":10,"offset":14}},` +
		`"message":"skill frontmatter is missing \"name\""}`
	if string(got) != want {
		t.Errorf("Diagnostic JSON =\n  %s\nwant\n  %s", got, want)
	}
}

func TestDiagnosticJSONWithDetail(t *testing.T) {
	d := Diagnostic{
		RuleID:   "skills/body-size",
		Severity: SeverityWarning,
		Path:     "skills/b/SKILL.md",
		Message:  "SKILL.md body exceeds configured word count",
		Detail:   "body has 1200 words; max is 1000",
	}

	got, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("json.Marshal(Diagnostic) = %v, want nil", err)
	}
	if !strings.Contains(string(got), `"detail":"body has 1200 words; max is 1000"`) {
		t.Errorf("JSON should contain detail, got %s", got)
	}
	if strings.Contains(string(got), `"fix"`) {
		t.Errorf("JSON should omit nil Fix, got %s", got)
	}
}
