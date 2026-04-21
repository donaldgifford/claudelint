// Package reporter renders aggregated Diagnostics to a target writer in
// one of the supported output formats. Phase 1.1 ships only the text
// formatter; JSON and GitHub Actions formats land in phase 1.7.
//
// Reporters are stateless by design — Format takes a Summary and writes
// once. This keeps the CLI wiring trivial (pick a reporter, hand it the
// summary, done) and the formats independent of each other.
package reporter

import (
	"fmt"
	"io"

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

// Text renders a Summary as human-readable text. The output shape is:
//
//	<path>:<line>:<col>: <severity>: <message> [<rule_id>]
//	...
//	<N> diagnostics, <M> files checked
//
// The trailing count line is always present so scripts can grep it
// even on zero-diagnostic runs. When a diagnostic has no position
// (file-level parse error), the line:col columns are omitted.
func Text(w io.Writer, s Summary) error {
	for i := range s.Diagnostics {
		if err := writeTextLine(w, &s.Diagnostics[i]); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "%d diagnostics, %d files checked\n", len(s.Diagnostics), s.Files)
	return err
}

func writeTextLine(w io.Writer, d *diag.Diagnostic) error {
	if d.Range.Start.IsZero() {
		_, err := fmt.Fprintf(
			w, "%s: %s: %s [%s]\n",
			d.Path, d.Severity, d.Message, d.RuleID,
		)
		return err
	}
	_, err := fmt.Fprintf(
		w, "%s:%d:%d: %s: %s [%s]\n",
		d.Path, d.Range.Start.Line, d.Range.Start.Column,
		d.Severity, d.Message, d.RuleID,
	)
	return err
}
