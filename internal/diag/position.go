package diag

// Position is a single point in a source file. Line and Column are
// 1-based so the output matches editor conventions; Offset is a 0-based
// byte index for tooling that wants to slice back into Artifact.Source.
// The zero value is a "no position" marker — reporters render it as the
// file itself (line 1, column 1) rather than a bogus location.
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Offset int `json:"offset"`
}

// IsZero reports whether the position carries no location information.
// Reporters use this to decide between file-level and inline rendering.
func (p Position) IsZero() bool {
	return p.Line == 0 && p.Column == 0 && p.Offset == 0
}

// Range is a half-open [Start, End) span within a source file. When
// End.IsZero() the diagnostic applies to a single point (typically a
// frontmatter key); when Start.IsZero() the diagnostic applies to the
// file as a whole (parse errors on malformed input).
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// IsZero reports whether the range carries no location information.
func (r Range) IsZero() bool {
	return r.Start.IsZero() && r.End.IsZero()
}

// PointRange constructs a Range that covers a single Position. It is
// the common case for diagnostics that target a specific frontmatter
// key or JSON value rather than a span.
func PointRange(p Position) Range {
	return Range{Start: p, End: p}
}
