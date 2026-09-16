package upstream

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Golden fixtures are section content, not section bytes.
//
// The documentation pages are large: the hook events section alone is
// 172 KB of prose, examples, and fifty-seven tables, of which one
// extractor reads only the headings. Committing that verbatim would put
// a megabyte of upstream prose in testdata and make a fixture refresh
// unreviewable. A snippet therefore keeps the structure the extractors
// read, headings and table rows, and drops the prose between them.
//
// Trimming is keyed by fixture path, not by extractor, because
// extractors that share an anchor share a file: the marketplace schema
// feeds three extractors, one of which reads a prose paragraph the
// other two do not. One rule per file is the only way that stays
// consistent.
//
// The rules live here rather than in a one-off script so the committed
// fixtures are exactly what "specdrift digest --write-snippets" writes,
// and refreshing them is a command rather than an act of judgement.

// maxSnippetCell caps a table cell in fixtures whose extractors read
// only a prefix of that cell.
const maxSnippetCell = 72

// mcpSnippetLines and changelogSnippetLines bound the two fixtures cut
// from whole pages rather than sections.
const (
	mcpSnippetLines       = 40
	changelogSnippetLines = 40
)

// snippetRules maps a fixture path to how that fixture is trimmed. A
// path with no rule keeps headings and table rows.
var snippetRules = map[string]func(string) string{
	// The event names are the headings; the fifty-seven tables under
	// this section belong to other extractors.
	"docs.hooks/hook-events.md": headingsOnly,

	// The reserved names live in a prose paragraph the default rule
	// would drop, alongside the two field tables.
	"docs.marketplaces/marketplace-schema.md": keepStructureAnd(reservedNamesParagraph),

	// These extractors read a field name, a type, and a requiredness,
	// never the full description, so the wide columns can be clipped.
	"docs.skills/frontmatter-reference.md":          truncateCells,
	"docs.skills/available-string-substitutions.md": truncateCells,
	"docs.plugins/plugin-manifest-schema.md":        truncateCells,
	"docs.plugins/file-locations-reference.md":      truncateCells,
	"docs.marketplaces/plugin-entries.md":           truncateCells,
	"docs.marketplaces/plugin-sources.md":           truncateCells,
	"docs.tools/tools-reference.md":                 truncateCells,

	// Whole-page extractors: keep the lines that carry the facts.
	"docs.mcp/docs.mcp.md":                         mcpTransportLines,
	"claudecode.changelog/claudecode.changelog.md": changelogHead,

	// JSON Schema documents are mostly prose keywords.
	"schemastore.plugin/schemastore.plugin.json":           pruneSchema,
	"schemastore.marketplace/schemastore.marketplace.json": pruneSchema,
	"schemastore.settings/schemastore.settings.json":       pruneSchema,
}

func mcpTransportLines(section string) string {
	return linesMatching(section, mcpTypeLiteral, mcpSnippetLines)
}

func changelogHead(section string) string {
	return headLines(section, changelogSnippetLines)
}

// Snippet returns the golden fixture content for one extractor.
func Snippet(e Extractor, section string) string {
	if rule, ok := snippetRules[SnippetPath(e)]; ok {
		return rule(section)
	}
	if e.Anchor() == nil {
		return section
	}

	return headingsAndTables(section)
}

// SnippetPath is the fixture path for an extractor, relative to the
// snippets directory. Extractors that share an anchor share a file, and
// extractors that read a whole page share one file per source.
func SnippetPath(e Extractor) string {
	src, _ := SourceByID(e.Source())

	ext := src.Ext
	if ext == "" {
		ext = extMarkdown
	}

	stem := e.Source()
	if a := e.Anchor(); a != nil {
		stem = anchorSlug(a)
	}

	return path.Join(e.Source(), stem+"."+ext)
}

// anchorRegexSyntax matches the regular-expression scaffolding around
// an anchor's literal heading text.
var anchorRegexSyntax = regexp.MustCompile(`\(\?m\)|\\s\*|\^|\$|\\`)

// nonSlugChar matches every run that is not slug-safe.
var nonSlugChar = regexp.MustCompile(`[^a-z0-9]+`)

// anchorSlug turns an anchor pattern into a file stem.
func anchorSlug(a *regexp.Regexp) string {
	s := anchorRegexSyntax.ReplaceAllString(a.String(), "")
	s = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(s), "#"))
	s = nonSlugChar.ReplaceAllString(strings.ToLower(s), "-")

	return strings.Trim(s, "-")
}

// headingsAndTables keeps every heading and every table row, dropping
// prose and fenced blocks.
func headingsAndTables(section string) string {
	return keepLines(section, func(ln string) bool {
		return headingLevel(ln) != 0 || isRow(ln)
	})
}

// headingsOnly keeps the headings and drops everything else.
func headingsOnly(section string) string {
	return keepLines(section, func(ln string) bool { return headingLevel(ln) != 0 })
}

// keepStructureAnd keeps headings, table rows, and any line matching re.
func keepStructureAnd(re *regexp.Regexp) func(string) string {
	return func(section string) string {
		return keepLines(section, func(ln string) bool {
			return headingLevel(ln) != 0 || isRow(ln) || re.MatchString(ln)
		})
	}
}

// keepLines filters a section line by line, skipping fenced blocks.
//
// Where it drops lines it leaves a blank one behind. Two tables
// separated only by prose would otherwise end up adjacent, and a GFM
// table runs until a line that is not a row: the second table's header
// would be read as a body row of the first.
func keepLines(section string, keep func(string) bool) string {
	var (
		out     []string
		fence   fenceState
		dropped bool
	)

	for _, ln := range strings.Split(section, "\n") {
		if fence.step(ln) {
			dropped = true

			continue
		}
		if !keep(ln) {
			dropped = true

			continue
		}

		if dropped && len(out) > 0 {
			out = append(out, "")
		}
		dropped = false

		out = append(out, strings.TrimRight(ln, " \t"))
	}

	return strings.Join(out, "\n") + "\n"
}

// truncateCells keeps headings and tables, then clips every cell after
// the first. It is only used where the extractor reads a prefix of
// those cells, never where enum values or numbers hide in the prose.
func truncateCells(section string) string {
	var out []string

	for _, ln := range strings.Split(headingsAndTables(section), "\n") {
		cells := splitRow(ln)
		if !isRow(ln) || isDelimiterRow(ln, cells, len(cells)) {
			out = append(out, ln)

			continue
		}

		for i := 1; i < len(cells); i++ {
			cells[i] = clip(cells[i], maxSnippetCell)
		}
		out = append(out, "| "+strings.Join(cells, " | ")+" |")
	}

	return strings.Join(out, "\n")
}

// clip shortens s to at most n bytes without leaving an unclosed code
// span or a dangling escape behind.
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}

	s = strings.TrimRight(s[:n], `\`)
	if strings.Count(s, "`")%2 != 0 {
		s = s[:strings.LastIndex(s, "`")]
	}

	return strings.TrimSpace(s)
}

// linesMatching keeps the page title and the first limit lines that
// match re, which is how a fixture for a whole-page extractor stays
// small.
func linesMatching(section string, re *regexp.Regexp, limit int) string {
	var out []string

	for _, ln := range strings.Split(section, "\n") {
		if len(out) >= limit {
			break
		}
		if len(out) == 0 && headingLevel(ln) == 1 {
			out = append(out, strings.TrimRight(ln, " \t"), "")

			continue
		}
		if re.MatchString(ln) {
			out = append(out, strings.TrimSpace(ln))
		}
	}

	return strings.Join(out, "\n") + "\n"
}

// headLines keeps the first n lines of a section.
func headLines(section string, n int) string {
	lines := strings.Split(section, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}

	return strings.Join(lines, "\n") + "\n"
}

// schemaProseKeys are the JSON Schema keywords that carry documentation
// rather than structure. Dropping them cuts the settings schema from
// 230 KB to a reviewable fixture without touching any value an
// extractor reads.
var schemaProseKeys = map[string]struct{}{
	"$comment":            {},
	"default":             {},
	"deprecated":          {},
	"description":         {},
	"errorMessage":        {},
	"examples":            {},
	"markdownDescription": {},
	"title":               {},
}

// pruneSchema removes the prose keywords from a JSON Schema document. A
// document it cannot parse is returned unchanged, so a fixture is never
// silently emptied.
func pruneSchema(section string) string {
	var root any
	if err := json.Unmarshal([]byte(section), &root); err != nil {
		return section
	}

	return encodePruned(section, collapse(prune(root), 0))
}

// encodePruned renders a pruned document, falling back to the original
// text if it cannot be encoded.
func encodePruned(section string, pruned any) string {
	var b strings.Builder
	if err := encodeJSON(&b, pruned); err != nil {
		return section
	}

	return b.String()
}

// schemaKeepDepth is how deep a fixture keeps full sub-schemas. Below
// it, a branch survives only if it still carries a value an extractor
// reads.
const schemaKeepDepth = 3

// collapse replaces every deep sub-schema that carries no enum or const
// with an empty object, keeping its key. The extractors read top-level
// property names and the enum and const values buried in anyOf
// branches; everything between the two is repetition that would triple
// the fixture.
func collapse(node any, depth int) any {
	switch v := node.(type) {
	case map[string]any:
		if depth >= schemaKeepDepth && !carriesSchemaValue(v) {
			// Keep the keys, drop the sub-schemas. Several extractors
			// read property names at depths the schemas reach only
			// through $ref, so emptying the object outright would
			// silently gut the fixture.
			stub := make(map[string]any, len(v))
			for k := range v {
				stub[k] = map[string]any{}
			}

			return stub
		}

		out := make(map[string]any, len(v))
		for k, child := range v {
			out[k] = collapse(child, depth+1)
		}

		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, child := range v {
			out = append(out, collapse(child, depth+1))
		}

		return out
	default:
		return node
	}
}

// carriesSchemaValue reports whether a subtree contains an enum or a
// const anywhere inside it.
func carriesSchemaValue(node any) bool {
	found := false

	walkJSON(node, nil, func(path []string, _ any) {
		if found || len(path) == 0 {
			return
		}
		switch path[len(path)-1] {
		case jsonKeywordEnum, jsonKeywordConst:
			found = true
		}
	})

	return found
}

// schemaNameMaps are the keywords whose children are user-chosen names
// rather than schema keywords. A property really can be called
// "description", so pruning by name inside one of these would delete a
// field the extractors count.
var schemaNameMaps = map[string]struct{}{
	"$defs":             {},
	"definitions":       {},
	"patternProperties": {},
	"properties":        {},
}

// prune drops the documentation keywords from a decoded schema.
func prune(node any) any { return pruneNode(node, false) }

func pruneNode(node any, named bool) any {
	switch v := node.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, child := range v {
			if _, drop := schemaProseKeys[k]; drop && !named {
				continue
			}
			_, isNameMap := schemaNameMaps[k]
			out[k] = pruneNode(child, isNameMap)
		}

		return out
	case []any:
		out := make([]any, 0, len(v))
		for _, child := range v {
			out = append(out, pruneNode(child, false))
		}

		return out
	default:
		return node
	}
}

// WriteSnippets writes the fixture files produced by an Extract run
// into dir, creating one subdirectory per source.
func WriteSnippets(dir string, snippets map[string]string) error {
	for rel, content := range snippets {
		full := filepath.Join(dir, filepath.FromSlash(rel))

		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			return fmt.Errorf("create snippet directory: %w", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write snippet %s: %w", rel, err)
		}
	}

	return nil
}
