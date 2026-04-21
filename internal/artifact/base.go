package artifact

import "github.com/donaldgifford/claudelint/internal/diag"

// Base is the common state every concrete Artifact embeds. It holds
// the path, the raw source bytes, and a byte-offset line index so
// parsers and rules can convert positions between line/column and
// offset without having to re-scan the source.
//
// The fields are unexported; access goes through Path, Source, and
// ResolvePosition so Base's invariants (line index in sync with
// source) stay internal to the package.
type Base struct {
	path      string
	source    []byte
	lineIndex []int
}

// NewBase builds a Base from a path and raw source, precomputing the
// line-start offset table. lineIndex[i] is the byte offset of the
// start of line (i+1); lineIndex[0] is always 0.
func NewBase(path string, source []byte) Base {
	idx := []int{0}
	for i, b := range source {
		if b == '\n' {
			idx = append(idx, i+1)
		}
	}
	out := make([]byte, len(source))
	copy(out, source)
	return Base{path: path, source: out, lineIndex: idx}
}

// Path returns the repo-relative path of this artifact.
func (b *Base) Path() string { return b.path }

// Source returns the raw bytes the parser consumed. Callers must not
// mutate the returned slice; it is the exact input a reporter may
// slice for snippets.
func (b *Base) Source() []byte { return b.source }

// ResolvePosition converts a 0-based byte offset into a 1-based
// line/column Position. Out-of-range offsets clamp to the end of the
// source so buggy callers cannot panic reporters.
func (b *Base) ResolvePosition(offset int) diag.Position {
	switch {
	case offset < 0:
		offset = 0
	case offset > len(b.source):
		offset = len(b.source)
	}

	line := searchLine(b.lineIndex, offset)
	col := offset - b.lineIndex[line-1] + 1
	return diag.Position{Line: line, Column: col, Offset: offset}
}

// ResolveRange is a convenience that converts a [start, end) byte
// range into a diag.Range.
func (b *Base) ResolveRange(start, end int) diag.Range {
	return diag.Range{Start: b.ResolvePosition(start), End: b.ResolvePosition(end)}
}

// searchLine returns the 1-based line number whose start offset is
// the largest value in idx not exceeding offset. idx must be
// non-empty and sorted (newBase guarantees both).
func searchLine(idx []int, offset int) int {
	lo, hi := 0, len(idx)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if idx[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1
}
