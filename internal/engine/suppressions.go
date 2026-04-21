package engine

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
)

// suppressionMarker matches a single claudelint in-source suppression
// inside an HTML comment. Two flavours are recognized:
//
//	<!-- claudelint:ignore=<id>[,<id>...] -->       per-line
//	<!-- claudelint:ignore-file=<id>[,<id>...] -->  whole-file
//
// Whitespace around the `=` and between comma-separated IDs is
// tolerated; whitespace inside IDs is not (they are stable
// "category/name" tokens).
var suppressionMarker = regexp.MustCompile(
	`<!--\s*claudelint:(ignore|ignore-file)\s*=\s*([A-Za-z0-9/_,\-\s]+?)\s*-->`,
)

// suppressor carries the parsed suppression data for a single
// Markdown artifact. It is populated once per run and consulted by
// the engine when filtering diagnostics.
type suppressor struct {
	// perLine[n] is the set of rule IDs suppressed on file line n.
	// A zero-value nil map means "no per-line suppressions".
	perLine map[int]map[string]struct{}
	// perFile is the set of rule IDs suppressed for the whole file.
	perFile map[string]struct{}
}

// newSuppressor parses a (Markdown) artifact's source and extracts
// every recognized claudelint suppression marker. Non-Markdown
// artifacts (JSON hook files, plugin manifests) produce an empty
// suppressor — those kinds rely on config-level suppression.
func newSuppressor(a artifact.Artifact) *suppressor {
	s := &suppressor{
		perFile: make(map[string]struct{}),
		perLine: make(map[int]map[string]struct{}),
	}
	if !supportsInSourceSuppression(a.Kind()) {
		return s
	}
	src := a.Source()
	for _, m := range suppressionMarker.FindAllSubmatchIndex(src, -1) {
		flavour := string(src[m[2]:m[3]])
		ids := splitIDs(string(src[m[4]:m[5]]))
		if flavour == "ignore-file" {
			for _, id := range ids {
				s.perFile[id] = struct{}{}
			}
			continue
		}
		line := lineOf(src, m[0])
		if s.perLine[line] == nil {
			s.perLine[line] = make(map[string]struct{})
		}
		for _, id := range ids {
			s.perLine[line][id] = struct{}{}
		}
		// The marker can also appear above the line it guards; add
		// the *next* non-blank line too.
		next := nextNonBlankLine(src, m[1])
		if next > 0 {
			if s.perLine[next] == nil {
				s.perLine[next] = make(map[string]struct{})
			}
			for _, id := range ids {
				s.perLine[next][id] = struct{}{}
			}
		}
	}
	return s
}

// suppressed reports whether d is silenced for the artifact a.
// Callers already filter by Path, so this function only consults the
// artifact-local suppressor.
func (s *suppressor) suppressed(d *diag.Diagnostic) bool {
	if _, ok := s.perFile[d.RuleID]; ok {
		return true
	}
	if line := d.Range.Start.Line; line > 0 {
		if set, ok := s.perLine[line]; ok {
			if _, hit := set[d.RuleID]; hit {
				return true
			}
		}
	}
	return false
}

// supportsInSourceSuppression reports whether in-source suppression
// comments are recognized for this artifact kind. Markdown-based
// artifacts support it; JSON-based artifacts rely on config-level
// suppression (JSON has no standard comment syntax).
func supportsInSourceSuppression(k artifact.ArtifactKind) bool {
	switch k {
	case artifact.KindClaudeMD, artifact.KindSkill, artifact.KindCommand, artifact.KindAgent:
		return true
	}
	return false
}

// splitIDs turns a comma-separated list into a clean []string.
func splitIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if id := strings.TrimSpace(p); id != "" {
			out = append(out, id)
		}
	}
	return out
}

// lineOf returns the 1-based line number containing byte offset off.
func lineOf(src []byte, off int) int {
	if off < 0 || off > len(src) {
		return 0
	}
	return bytes.Count(src[:off], []byte("\n")) + 1
}

// nextNonBlankLine returns the 1-based line number of the first
// non-blank line *after* the byte offset end. Returns 0 when there
// is no such line.
func nextNonBlankLine(src []byte, end int) int {
	if end >= len(src) {
		return 0
	}
	// Walk forward to the end of the current line.
	i := end
	for i < len(src) && src[i] != '\n' {
		i++
	}
	line := lineOf(src, i) + 1
	for i < len(src) {
		i++
		// Line start: trim leading whitespace.
		start := i
		for i < len(src) && src[i] != '\n' {
			i++
		}
		if len(bytes.TrimSpace(src[start:i])) > 0 {
			return line
		}
		line++
	}
	return 0
}
