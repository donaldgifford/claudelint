// Package diag defines the diagnostic vocabulary shared by rules, the
// engine, and reporters. It is the narrowest useful shape — a rule
// produces Diagnostics; the engine sorts and filters them; reporters
// marshal them. No package outside diag should define its own Severity
// or Range.
package diag

import (
	"encoding"
	"fmt"
)

// Severity classifies a Diagnostic. The zero value is SeverityInfo so
// an unset severity never masquerades as an error.
type Severity int

const (
	// SeverityInfo is advisory output that does not affect the exit
	// code. It is the zero value so a forgotten assignment cannot
	// accidentally flag an error.
	SeverityInfo Severity = iota

	// SeverityWarning is a style or soundness concern. Warnings do not
	// fail the build by default; --max-warnings=N can promote overflow
	// to a hard failure.
	SeverityWarning

	// SeverityError fails the run. Reserved for problems the user
	// unambiguously must fix (parse errors, required fields missing,
	// unknown hook events).
	SeverityError
)

// String returns the canonical lowercase name of the severity, suitable
// for JSON, text, and GitHub Actions output formats.
func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return fmt.Sprintf("severity(%d)", int(s))
	}
}

// MarshalText implements encoding.TextMarshaler so Severity encodes as
// "info" / "warning" / "error" in JSON and YAML.
func (s Severity) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText accepts the canonical lowercase names and rejects
// anything else. Config files and test fixtures go through this path,
// so being strict keeps typos from silently downgrading severities.
func (s *Severity) UnmarshalText(text []byte) error {
	switch string(text) {
	case "info":
		*s = SeverityInfo
	case "warning":
		*s = SeverityWarning
	case "error":
		*s = SeverityError
	default:
		return fmt.Errorf("unknown severity %q (want info, warning, or error)", string(text))
	}
	return nil
}

var (
	_ encoding.TextMarshaler   = Severity(0)
	_ encoding.TextUnmarshaler = (*Severity)(nil)
)
