package claudemd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&importExists{}) }

// maxImportHops is the documented recursive import depth — imported
// files can recursively import other files, up to four hops.
const maxImportHops = 4

// categoryContent is shared by this package's content rules (goconst).
const categoryContent = "content"

// importExists warns when a CLAUDE.md `@path` import does not resolve
// on disk, and when an import chain exceeds the documented four-hop
// depth. A missing import silently loads nothing; content past the
// depth cap silently never enters context.
//
// Per the documented parser behavior, imports inside Markdown code
// spans (`@README`) and fenced code blocks are skipped. Relative
// paths resolve against the file containing the import (not the
// working directory); `~/` resolves against the home directory.
//
// Like mcp/command-exists-on-path, this rule intentionally touches
// the filesystem — existence is the property under test.
type importExists struct{}

func (*importExists) ID() string                     { return "claude_md/import-exists" }
func (*importExists) Category() string               { return categoryContent }
func (*importExists) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*importExists) DefaultOptions() map[string]any { return nil }
func (*importExists) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindClaudeMD}
}

func (*importExists) HelpURI() string { return rules.DefaultHelpURI("claude_md/import-exists") }

func (r *importExists) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	c, ok := a.(*artifact.ClaudeMD)
	if !ok {
		return nil
	}
	src := c.Source()
	baseDir := filepath.Dir(c.Path())

	var out []diag.Diagnostic
	for _, imp := range scanImports(src) {
		rng := artifact.ResolveOffsetRange(src, imp.start, imp.end)
		target, terr := resolveImportPath(baseDir, imp.path)
		if terr != nil {
			// Unresolvable home directory — nothing to check.
			continue
		}
		if _, err := os.Stat(target); err != nil {
			out = append(out, diag.Diagnostic{
				RuleID:  r.ID(),
				Path:    c.Path(),
				Range:   rng,
				Message: fmt.Sprintf("import @%s does not resolve (%s)", imp.path, target),
			})
			continue
		}
		if chain := deepestChain(target, map[string]bool{target: true}); 1+chain > maxImportHops {
			out = append(out, diag.Diagnostic{
				RuleID: r.ID(),
				Path:   c.Path(),
				Range:  rng,
				Message: fmt.Sprintf(
					"import @%s starts a chain deeper than the documented %d-hop limit — the tail never loads",
					imp.path, maxImportHops),
			})
		}
	}
	return out
}

// importRef is one @path occurrence with its byte offsets in the
// scanned source.
type importRef struct {
	path       string
	start, end int
}

// importPattern matches an @path token at a word boundary. The
// preceding character must not be alphanumeric so emails
// (user@host) don't count.
var importPattern = regexp.MustCompile(`(^|[^A-Za-z0-9_` + "`" + `])@([~A-Za-z0-9_./-]+)`)

// scanImports extracts @path imports from src, skipping fenced code
// blocks and inline code spans per the documented parser behavior.
func scanImports(src []byte) []importRef {
	var out []importRef
	offset := 0
	inFence := false
	for _, line := range strings.SplitAfter(string(src), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			offset += len(line)
			continue
		}
		if !inFence {
			out = append(out, lineImports(line, offset)...)
		}
		offset += len(line)
	}
	return out
}

// lineImports scans a single non-fenced line, masking inline code
// spans so `@README` stays literal.
func lineImports(line string, offset int) []importRef {
	masked := maskCodeSpans(line)
	var out []importRef
	for _, m := range importPattern.FindAllStringSubmatchIndex(masked, -1) {
		pathStart, pathEnd := m[4], m[5]
		path := strings.TrimRight(masked[pathStart:pathEnd], ".,;:")
		if path == "" || path == "~" {
			continue
		}
		out = append(out, importRef{
			path:  path,
			start: offset + pathStart - 1, // include the @
			end:   offset + pathStart + len(path),
		})
	}
	return out
}

// maskCodeSpans replaces backtick-delimited spans with spaces of the
// same width so offsets stay aligned while their content is ignored.
func maskCodeSpans(line string) string {
	b := []byte(line)
	for pos := 0; pos < len(b); {
		open := strings.IndexByte(string(b[pos:]), '`')
		if open < 0 {
			break
		}
		open += pos
		closing := strings.IndexByte(string(b[open+1:]), '`')
		end := len(b) - 1
		if closing >= 0 {
			end = open + 1 + closing
		}
		// Unbalanced backtick masks through end of line.
		for i := open; i <= end; i++ {
			b[i] = ' '
		}
		pos = end + 1
	}
	return string(b)
}

// resolveImportPath resolves an import target: `~/` against the home
// directory, absolute paths as-is, everything else against the
// importing file's directory.
func resolveImportPath(baseDir, path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Join(baseDir, path), nil
}

// deepestChain returns the longest import-hop count reachable from
// file (0 when it imports nothing). visited guards cycles; the walk
// is capped at the documented limit plus one since deeper chains are
// all equally broken.
func deepestChain(file string, visited map[string]bool) int {
	if len(visited) > maxImportHops+1 {
		return maxImportHops + 1
	}
	src, err := os.ReadFile(file)
	if err != nil {
		return 0
	}
	deepest := 0
	baseDir := filepath.Dir(file)
	for _, imp := range scanImports(src) {
		target, terr := resolveImportPath(baseDir, imp.path)
		if terr != nil || visited[target] {
			continue
		}
		visited[target] = true
		if d := 1 + deepestChain(target, visited); d > deepest {
			deepest = d
		}
		delete(visited, target)
	}
	return deepest
}
