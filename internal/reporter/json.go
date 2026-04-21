package reporter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// JSONSchemaVersion is bumped whenever the JSON report shape changes
// in a way that external consumers need to notice. v1 is the shape
// documented in docs/reference/json-schema.md. Golden tests compare
// against this exact value.
const JSONSchemaVersion = "1"

// jsonReport is the top-level JSON shape. Fields are exported so
// encoding/json picks them up via struct tags, but the type itself is
// unexported — the wire contract is the JSON fields, not the Go
// struct.
type jsonReport struct {
	SchemaVersion   string            `json:"schema_version"`
	RulesetVersion  string            `json:"ruleset_version"`
	Fingerprint     string            `json:"fingerprint"`
	FilesChecked    int               `json:"files_checked"`
	DiagnosticCount int               `json:"diagnostic_count"`
	SeverityCount   jsonSeverityCount `json:"severity_count"`
	Diagnostics     []diag.Diagnostic `json:"diagnostics"`
}

type jsonSeverityCount struct {
	Error   int `json:"error"`
	Warning int `json:"warning"`
	Info    int `json:"info"`
}

// JSON renders a Summary as a stable JSON object to w. The top-level
// object always contains the same keys, even when diagnostics is empty —
// consumers can rely on `severity_count.error` existing without a
// presence check. Output ends with a trailing newline so pipelines
// that expect it (shell, jq) stay happy.
func JSON(w io.Writer, s Summary) error {
	report := jsonReport{
		SchemaVersion:   JSONSchemaVersion,
		RulesetVersion:  rules.RulesetVersion,
		Fingerprint:     rules.RulesetFingerprint(),
		FilesChecked:    s.Files,
		DiagnosticCount: len(s.Diagnostics),
		SeverityCount:   countSeverities(s.Diagnostics),
		Diagnostics:     ensureNonNil(s.Diagnostics),
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode json report: %w", err)
	}
	return nil
}

// ensureNonNil replaces a nil slice with an empty one so JSON encodes
// `[]` rather than `null`. Consumers have an easier time with the
// empty array.
func ensureNonNil(d []diag.Diagnostic) []diag.Diagnostic {
	if d == nil {
		return []diag.Diagnostic{}
	}
	return d
}

func countSeverities(ds []diag.Diagnostic) jsonSeverityCount {
	var c jsonSeverityCount
	for i := range ds {
		switch ds[i].Severity {
		case diag.SeverityError:
			c.Error++
		case diag.SeverityWarning:
			c.Warning++
		case diag.SeverityInfo:
			c.Info++
		}
	}
	return c
}
