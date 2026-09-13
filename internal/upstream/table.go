package upstream

import (
	"regexp"
	"strings"
)

// The upstream pages are Mintlify Markdown exports: CommonMark plus GFM
// tables, with fenced code blocks that frequently contain shell
// comments. Those comment lines look exactly like ATX headings, so
// every scan in this file tracks fence state; a whole-page regexp would
// stop a section early at the first "# Add to deny rules:" in a bash
// block.
//
// Nothing here is a general Markdown parser. It reads headings, table
// rows, and backticked tokens, which is all the extractors need, and it
// adds no dependency: goldmark would be a new module for one construct.

// maxHeadingLevel is the deepest ATX heading CommonMark allows.
const maxHeadingLevel = 6

// maxIndent is the deepest indent at which a line is still a heading or
// a fence rather than an indented code block.
const maxIndent = 3

// backtickedToken matches one inline code span. Cells in the upstream
// tables wrap field names, enum values, and reserved names in single
// backticks.
var backtickedToken = regexp.MustCompile("`([^`]+)`")

// Table is a parsed GFM table. Cells keep their source text, trimmed of
// surrounding whitespace and with escaped pipes resolved; use Name for
// the backtick-stripped form of an identifier cell.
type Table struct {
	// Header is the column titles, lowercased and trimmed.
	Header []string
	// Rows is the body rows. Short rows are padded to len(Header) so a
	// column index is always safe.
	Rows [][]string
}

// Column returns the index of the named header, or -1. The comparison
// is case-insensitive and ignores surrounding whitespace, because the
// upstream tables are not consistent about either.
func (t Table) Column(name string) int {
	want := strings.ToLower(strings.TrimSpace(name))
	for i, h := range t.Header {
		if h == want {
			return i
		}
	}

	return -1
}

// HasHeaders reports whether the table's first len(want) columns are
// exactly want. Extractors call this so an upstream column rename is a
// loud failure rather than a silently empty section.
func (t Table) HasHeaders(want ...string) bool {
	if len(t.Header) < len(want) {
		return false
	}
	for i, w := range want {
		if t.Header[i] != strings.ToLower(strings.TrimSpace(w)) {
			return false
		}
	}

	return true
}

// Cell returns row[col], or "" when either index is out of range.
func (t Table) Cell(row, col int) string {
	if row < 0 || row >= len(t.Rows) {
		return ""
	}
	if col < 0 || col >= len(t.Rows[row]) {
		return ""
	}

	return t.Rows[row][col]
}

// Section returns the lines of page from the first heading matching
// anchor up to, but not including, the next heading of equal or higher
// level. The anchor heading itself is included so a written-out snippet
// says what it is. Headings inside fenced code blocks are ignored.
func Section(page []byte, anchor *regexp.Regexp) (string, bool) {
	lines := splitLines(page)

	start, level := -1, 0
	var fence fenceState

	for i, ln := range lines {
		if fence.step(ln) {
			continue
		}

		lv := headingLevel(ln)
		if lv == 0 {
			continue
		}

		if start < 0 {
			if anchor.MatchString(ln) {
				start, level = i, lv
			}

			continue
		}

		if lv <= level {
			return strings.Join(lines[start:i], "\n") + "\n", true
		}
	}

	if start < 0 {
		return "", false
	}

	return strings.Join(lines[start:], "\n") + "\n", true
}

// Headings returns the text of every heading at the given level in src,
// in document order, with the leading hashes and surrounding whitespace
// removed. Headings inside fenced code blocks are ignored.
func Headings(src string, level int) []string {
	var (
		out   []string
		fence fenceState
	)

	for _, ln := range strings.Split(src, "\n") {
		if fence.step(ln) {
			continue
		}
		if headingLevel(ln) != level {
			continue
		}
		out = append(out, strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(ln), "#")))
	}

	return out
}

// Tables returns every GFM table in src, in document order. Tables
// inside fenced code blocks are ignored.
func Tables(src string) []Table {
	lines := strings.Split(src, "\n")

	var (
		out   []Table
		fence fenceState
	)

	for i := 0; i < len(lines); i++ {
		if fence.step(lines[i]) {
			continue
		}

		tbl, next, ok := tableAt(lines, i)
		if !ok {
			continue
		}
		out = append(out, tbl)
		i = next - 1
	}

	return out
}

// FirstTable returns the first table in src. ok is false when there is
// none.
func FirstTable(src string) (Table, bool) {
	tables := Tables(src)
	if len(tables) == 0 {
		return Table{}, false
	}

	return tables[0], true
}

// tableAt parses a table whose header row is lines[i]. next is the
// index of the first line after the table.
func tableAt(lines []string, i int) (Table, int, bool) {
	if i+1 >= len(lines) || !isRow(lines[i]) {
		return Table{}, i, false
	}

	header := splitRow(lines[i])
	delim := splitRow(lines[i+1])

	if !isDelimiterRow(lines[i+1], delim, len(header)) {
		return Table{}, i, false
	}

	tbl := Table{Header: make([]string, len(header))}
	for j, h := range header {
		tbl.Header[j] = strings.ToLower(strings.TrimSpace(h))
	}

	j := i + 2
	for ; j < len(lines) && isRow(lines[j]); j++ {
		cells := splitRow(lines[j])
		for len(cells) < len(tbl.Header) {
			cells = append(cells, "")
		}
		tbl.Rows = append(tbl.Rows, cells)
	}

	return tbl, j, true
}

// isRow reports whether a line could be a table row: it must contain an
// unescaped pipe and must not be a heading or blank.
func isRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || headingLevel(line) != 0 {
		return false
	}

	return strings.ContainsRune(stripEscapes(trimmed), '|')
}

// isDelimiterRow reports whether line is a GFM alignment row with the
// expected number of cells. Every cell must be dashes with optional
// leading and trailing colons.
func isDelimiterRow(line string, cells []string, want int) bool {
	if len(cells) != want || !strings.Contains(line, "-") {
		return false
	}

	for _, c := range cells {
		c = strings.TrimSpace(c)
		c = strings.TrimPrefix(c, ":")
		c = strings.TrimSuffix(c, ":")
		if c == "" || strings.Trim(c, "-") != "" {
			return false
		}
	}

	return true
}

// splitRow splits a table row on unescaped pipes, dropping the optional
// leading and trailing delimiters and trimming each cell. Escaped pipes
// are resolved to literal pipes.
func splitRow(line string) []string {
	s := strings.TrimSpace(line)

	var (
		cells   []string
		cur     strings.Builder
		escaped bool
	)

	for _, r := range s {
		switch {
		case escaped:
			if r != '|' {
				cur.WriteRune('\\')
			}
			cur.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '|':
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	if escaped {
		cur.WriteRune('\\')
	}
	cells = append(cells, strings.TrimSpace(cur.String()))

	// A leading pipe produces an empty first cell and a trailing pipe an
	// empty last one. Both are delimiters, not data.
	if len(cells) > 1 && strings.HasPrefix(s, "|") {
		cells = cells[1:]
	}
	if len(cells) > 1 && endsWithUnescapedPipe(s) {
		cells = cells[:len(cells)-1]
	}

	return cells
}

func endsWithUnescapedPipe(s string) bool {
	if !strings.HasSuffix(s, "|") {
		return false
	}

	// Count the backslashes before the final pipe; an odd run escapes it.
	n := 0
	for i := len(s) - 2; i >= 0 && s[i] == '\\'; i-- {
		n++
	}

	return n%2 == 0
}

// stripEscapes removes backslash escapes so a scan for structural
// characters does not see the escaped ones.
func stripEscapes(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}

	var (
		b       strings.Builder
		escaped bool
	)

	for _, r := range s {
		if escaped {
			escaped = false

			continue
		}
		if r == '\\' {
			escaped = true

			continue
		}
		b.WriteRune(r)
	}

	return b.String()
}

// Name strips surrounding inline-code backticks and whitespace from a
// cell, which is how every upstream table writes a field, enum, or tool
// identifier.
func Name(cell string) string {
	s := strings.TrimSpace(cell)
	for strings.HasPrefix(s, "`") && strings.HasSuffix(s, "`") && len(s) > 1 {
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(s, "`"), "`"))
	}

	return s
}

// BacktickedTokens returns every inline code span in s, in order, with
// duplicates preserved. Extractors use it to read enum values out of a
// description cell and names out of a prose paragraph.
func BacktickedTokens(s string) []string {
	matches := backtickedToken.FindAllStringSubmatch(s, -1)

	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if tok := strings.TrimSpace(m[1]); tok != "" {
			out = append(out, tok)
		}
	}

	return out
}

// splitLines splits page into lines, dropping carriage returns so a
// CRLF response never leaks into the digest.
func splitLines(page []byte) []string {
	s := strings.ReplaceAll(string(page), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	return strings.Split(s, "\n")
}

// headingLevel returns the ATX heading level of line, or 0 when the
// line is not a heading. CommonMark allows up to three leading spaces
// and requires a space or end of line after the hashes.
func headingLevel(line string) int {
	i := 0
	for i < len(line) && i < maxIndent && line[i] == ' ' {
		i++
	}

	n := 0
	for i+n < len(line) && line[i+n] == '#' {
		n++
	}

	if n == 0 || n > maxHeadingLevel {
		return 0
	}

	rest := line[i+n:]
	if rest != "" && !strings.HasPrefix(rest, " ") && !strings.HasPrefix(rest, "\t") {
		return 0
	}

	return n
}

// fenceState tracks whether the scan is inside a fenced code block.
type fenceState struct {
	open   bool
	marker byte
	length int
}

// step advances the state by one line and reports whether that line
// should be skipped: every line of a fenced block, including its two
// fences.
func (f *fenceState) step(line string) bool {
	marker, length, ok := fenceAt(line)

	switch {
	case !f.open && ok:
		f.open, f.marker, f.length = true, marker, length

		return true
	case f.open && ok && marker == f.marker && length >= f.length && fenceHasNoInfo(line):
		f.open = false

		return true
	}

	return f.open
}

// fenceAt reports the fence marker and run length at the start of line.
func fenceAt(line string) (byte, int, bool) {
	i := 0
	for i < len(line) && i < maxIndent && line[i] == ' ' {
		i++
	}
	if i >= len(line) || (line[i] != '`' && line[i] != '~') {
		return 0, 0, false
	}

	marker := line[i]

	n := 0
	for i+n < len(line) && line[i+n] == marker {
		n++
	}
	if n < 3 {
		return 0, 0, false
	}

	// An opening backtick fence may not carry a backtick in its info
	// string; that would make it an inline code span instead.
	if marker == '`' && strings.ContainsRune(line[i+n:], '`') {
		return 0, 0, false
	}

	return marker, n, true
}

// fenceHasNoInfo reports whether the fence line carries no info string,
// which a closing fence must not.
func fenceHasNoInfo(line string) bool {
	trimmed := strings.TrimSpace(line)

	return strings.Trim(trimmed, "`~") == ""
}
