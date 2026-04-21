// Package reporter renders aggregated Diagnostics to a target writer in
// one of the supported output formats. Phase 1.1 shipped the text
// formatter; phase 1.7 adds JSON and GitHub Actions formats and wires
// color / NO_COLOR into Text.
//
// Reporters are stateless by design — each Format function takes a
// Summary and writes once. This keeps the CLI wiring trivial (pick a
// reporter, hand it the summary, done) and the formats independent of
// each other.
package reporter

import (
	"fmt"
	"io"
	"os"

	"github.com/donaldgifford/claudelint/internal/diag"
)

// Summary is what the engine produces at the end of a run. The
// diagnostic list is already sorted and deduped; Files is the number of
// artifacts that were parsed and inspected, regardless of whether they
// produced diagnostics. Reporters use Files to render "N files checked"
// without the engine needing to know anything about output shape.
type Summary struct {
	Diagnostics []diag.Diagnostic
	Files       int
}

// TextOptions controls the human-readable text formatter. The zero
// value is valid: no color, no filtering.
type TextOptions struct {
	// Color enables ANSI color codes keyed to severity. The CLI toggles
	// this based on --no-color and the NO_COLOR env var.
	Color bool
}

// Text renders a Summary as human-readable text with default options
// (no color). Existing call sites keep compiling; new CLI code uses
// TextWithOptions for color control.
func Text(w io.Writer, s Summary) error {
	return TextWithOptions(w, s, TextOptions{})
}

// TextWithOptions renders a Summary as human-readable text. Output shape:
//
//	<path>:<line>:<col>: <severity>: <message> [<rule_id>]
//	...
//	<N> diagnostics, <M> files checked
//
// The trailing count line is always present so scripts can grep it
// even on zero-diagnostic runs. When a diagnostic has no position
// (file-level parse error), the line:col columns are omitted.
func TextWithOptions(w io.Writer, s Summary, opts TextOptions) error {
	for i := range s.Diagnostics {
		if err := writeTextLine(w, &s.Diagnostics[i], opts); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "%d diagnostics, %d files checked\n", len(s.Diagnostics), s.Files)
	return err
}

// ShouldUseColor returns true when the Text reporter should emit ANSI
// color codes. Precedence: explicit --no-color wins, then NO_COLOR env
// (per https://no-color.org), finally the caller's intent (usually
// "stdout is a TTY").
func ShouldUseColor(noColorFlag, intent bool) bool {
	if noColorFlag {
		return false
	}
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	return intent
}

func writeTextLine(w io.Writer, d *diag.Diagnostic, opts TextOptions) error {
	sev := colorizeSeverity(d.Severity, opts.Color)
	if d.Range.Start.IsZero() {
		_, err := fmt.Fprintf(
			w, "%s: %s: %s [%s]\n",
			d.Path, sev, d.Message, d.RuleID,
		)
		return err
	}
	_, err := fmt.Fprintf(
		w, "%s:%d:%d: %s: %s [%s]\n",
		d.Path, d.Range.Start.Line, d.Range.Start.Column,
		sev, d.Message, d.RuleID,
	)
	return err
}

// colorizeSeverity returns the severity label wrapped in ANSI color
// codes when color is enabled. Error→red, warning→yellow, info→cyan.
// The reset code always follows so the rest of the line is plain.
func colorizeSeverity(s diag.Severity, color bool) string {
	label := s.String()
	if !color {
		return label
	}
	const reset = "\x1b[0m"
	switch s {
	case diag.SeverityError:
		return "\x1b[31m" + label + reset
	case diag.SeverityWarning:
		return "\x1b[33m" + label + reset
	case diag.SeverityInfo:
		return "\x1b[36m" + label + reset
	default:
		return label
	}
}
