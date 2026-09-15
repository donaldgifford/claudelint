package upstream

import (
	"fmt"
	"regexp"
	"strings"
)

// Column headings the upstream field tables use.
const (
	colField    = "field"
	colRequired = "required"
	colType     = "type"
)

// Field groups used in plugins.manifest_fields.
const (
	groupRequired  = "required"
	groupMetadata  = "metadata"
	groupComponent = "component"
)

// mustAnchor compiles a heading pattern at package scope. Every anchor
// in this package is a literal, so a compile failure is a programming
// error, not a runtime one.
func mustAnchor(pattern string) *regexp.Regexp {
	return regexp.MustCompile(pattern)
}

// headingAnchor builds an anchor for an exact ATX heading at the given
// level.
func headingAnchor(level int, text string) *regexp.Regexp {
	return mustAnchor(`^` + strings.Repeat("#", level) + ` ` + regexp.QuoteMeta(text) + `\s*$`)
}

// firstTableWithHeaders returns the first table in section whose
// leading columns are want. It exists so an upstream column rename
// fails loudly instead of yielding an empty section.
func firstTableWithHeaders(section string, want ...string) (Table, error) {
	tables := Tables(section)
	if len(tables) == 0 {
		return Table{}, ErrTableNotFound
	}

	for _, t := range tables {
		if t.HasHeaders(want...) {
			return t, nil
		}
	}

	return Table{}, fmt.Errorf("%w: want leading columns %v, first table has %v",
		ErrHeaders, want, tables[0].Header)
}

// subSection scopes further inside an already-scoped section.
func subSection(section string, anchor *regexp.Regexp) (string, error) {
	sub, ok := Section([]byte(section), anchor)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrAnchorNotFound, anchor)
	}

	return sub, nil
}

// fieldRows turns a field table into digest rows. When the table has no
// "required" column, fallback supplies the value, which is how the
// documentation's "Required fields" and "Optional fields" subsections
// are read.
func fieldRows(t Table, group, fallback string) []Field {
	nameCol := t.Column(colField)
	if nameCol < 0 {
		nameCol = 0
	}
	reqCol := t.Column(colRequired)
	typeCol := t.Column(colType)

	out := make([]Field, 0, len(t.Rows))

	for i := range t.Rows {
		name := Name(t.Cell(i, nameCol))
		if name == "" {
			continue
		}

		f := Field{Name: name, Group: group, Required: fallback}
		if reqCol >= 0 {
			f.Required = requiredValue(t.Cell(i, reqCol))
		}
		if typeCol >= 0 {
			f.Type = Plain(t.Cell(i, typeCol))
		}

		out = append(out, f)
	}

	return out
}

// requiredValue normalises the requiredness column, which the pages
// write as Yes, No, or Recommended.
func requiredValue(cell string) string {
	switch strings.ToLower(Plain(cell)) {
	case "yes", "required", "true":
		return RequiredYes
	case "recommended":
		return RequiredRecommended
	case "":
		return ""
	default:
		return RequiredNo
	}
}

// Plain reduces a cell to its text: inline code spans, bold and italic
// markers, and Markdown link syntax are removed. Use it for values that
// are prose rather than identifiers.
func Plain(cell string) string {
	s := strings.TrimSpace(cell)
	s = markdownLink.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")

	return strings.TrimSpace(s)
}

// markdownLink matches an inline link, capturing its text.
var markdownLink = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

// unquote strips the surrounding double quotes the documentation puts
// around JSON string literals, such as `"command"`.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return s[1 : len(s)-1]
	}

	return s
}

// nonEmpty returns an ErrEmptySection error naming what was being read
// when v is empty. Extractors call it so a heading that survives an
// upstream rewrite but loses its content still fails the run.
func nonEmpty[T any](v []T, what string) error {
	if len(v) == 0 {
		return fmt.Errorf("%w: %s", ErrEmptySection, what)
	}

	return nil
}

// nonEmptyMap is nonEmpty for the map-valued sections.
func nonEmptyMap[K comparable, V any](m map[K]V, what string) error {
	if len(m) == 0 {
		return fmt.Errorf("%w: %s", ErrEmptySection, what)
	}

	return nil
}
