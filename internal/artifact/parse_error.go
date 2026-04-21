package artifact

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/diag"
)

// ParseError is the structured error every parser returns when the
// input cannot be turned into a typed Artifact. It carries the path
// of the offending file plus a diag.Range pointing at the byte span
// that triggered the failure so the engine can synthesize a
// schema/parse diagnostic without a rule having to inspect raw bytes.
//
// Parsers must always populate Path. Range may be zero when the
// failure is file-level (e.g. empty file, truncated YAML header) —
// reporters render a zero Range as "whole file".
type ParseError struct {
	// Path is the repo-relative path of the file that failed to parse.
	Path string

	// Range is the span within the file that the parser identifies as
	// the cause. A zero Range means the failure is file-level.
	Range diag.Range

	// Message is short, imperative, lowercase — the same style rules
	// use. It is what the schema/parse diagnostic ultimately renders.
	Message string

	// Cause is the underlying library error, retained via errors.As
	// for observability but never surfaced to end users directly.
	Cause error
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	if e.Range.IsZero() {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return fmt.Sprintf("%s:%d:%d: %s",
		e.Path, e.Range.Start.Line, e.Range.Start.Column, e.Message)
}

// Unwrap exposes Cause so errors.Is / errors.As traverse to the
// underlying library error.
func (e *ParseError) Unwrap() error { return e.Cause }
