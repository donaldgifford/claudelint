package artifact

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"

	"github.com/donaldgifford/claudelint/internal/diag"
)

// frontmatterFence is the delimiter YAML frontmatter uses (a line of
// exactly three dashes, terminated by \n). Whitespace-only fences like
// "--- \n" are tolerated by most Markdown front-matter parsers but we
// follow the strict form so diagnostics are unambiguous.
var frontmatterFence = []byte("---\n")

// markdownDoc is the shared output of a Markdown + frontmatter parse.
// Concrete parsers (ParseClaudeMD, ParseSkill, ParseCommand,
// ParseAgent) build their typed Artifact on top of this result.
type markdownDoc struct {
	Base        Base
	Frontmatter Frontmatter
	Body        diag.Range
	// fm is the decoded YAML map, nil when there is no frontmatter.
	fm map[string]any
}

// findCloseFence locates the YAML frontmatter close fence ("---"
// starting a line). It returns the slice indices for the YAML body
// start, body end, and close-fence end — exclusive of the trailing
// newline. found is false when no close fence exists, which the
// caller treats as an unterminated frontmatter parse error.
//
// Two close-fence forms are recognized:
//
//  1. "---\n...\n---[\n]" — the normal case.
//  2. "---\n---[\n]" — an empty frontmatter where the close follows
//     the opener immediately. The search below would otherwise miss
//     this because "\n---" appears only before the opener, not inside
//     the (empty) body.
func findCloseFence(src []byte, openEnd int) (bodyStart, bodyEnd, fenceEnd int, found bool) {
	bodyStart = openEnd

	// Immediate close: opener followed by "---" (with or without a
	// trailing newline) at openEnd.
	if openEnd+3 <= len(src) && bytes.Equal(src[openEnd:openEnd+3], []byte("---")) {
		if openEnd+3 == len(src) || src[openEnd+3] == '\n' {
			fenceEnd = openEnd + 3
			if fenceEnd < len(src) {
				fenceEnd++ // consume trailing \n
			}
			return openEnd, openEnd, fenceEnd, true
		}
	}

	idx := bytes.Index(src[openEnd:], []byte("\n---"))
	if idx < 0 {
		return 0, 0, 0, false
	}
	bodyEnd = openEnd + idx
	fenceEnd = bodyEnd + len("\n---")
	if fenceEnd < len(src) && src[fenceEnd] == '\n' {
		fenceEnd++
	}
	return bodyStart, bodyEnd, fenceEnd, true
}

// parseMarkdown splits src into its frontmatter block and body, parses
// the YAML frontmatter with goccy/go-yaml, and populates Frontmatter.
// Keys with byte-accurate ranges.
//
// A file with no frontmatter is accepted: Frontmatter.Block is zero
// and fm is nil. A file that opens with `---` but has no closing
// fence, or whose YAML does not parse, yields a *ParseError.
func parseMarkdown(path string, src []byte) (*markdownDoc, *ParseError) {
	base := NewBase(path, src)
	doc := &markdownDoc{Base: base}

	if !bytes.HasPrefix(src, frontmatterFence) {
		// No frontmatter. Body is the entire file.
		doc.Body = base.ResolveRange(0, len(src))
		return doc, nil
	}

	openEnd := len(frontmatterFence)
	fmBodyStart, fmBodyEnd, closeFenceEnd, found := findCloseFence(src, openEnd)
	if !found {
		return nil, &ParseError{
			Path:    path,
			Range:   base.ResolveRange(0, openEnd),
			Message: "unterminated YAML frontmatter (no closing ---)",
		}
	}

	fmSrc := src[fmBodyStart:fmBodyEnd]

	doc.Frontmatter.Block = base.ResolveRange(0, closeFenceEnd)
	doc.Body = base.ResolveRange(closeFenceEnd, len(src))

	if len(bytes.TrimSpace(fmSrc)) == 0 {
		// An empty frontmatter block parses fine but has no keys.
		doc.fm = map[string]any{}
		return doc, nil
	}

	file, err := parser.ParseBytes(fmSrc, parser.ParseComments)
	if err != nil {
		return nil, &ParseError{
			Path:    path,
			Range:   base.ResolveRange(fmBodyStart, fmBodyEnd),
			Message: fmt.Sprintf("invalid YAML frontmatter: %s", cleanYAMLError(err)),
			Cause:   err,
		}
	}

	keys, decoded, err := collectFrontmatterKeys(file, fmBodyStart, &base)
	if err != nil {
		return nil, &ParseError{
			Path:    path,
			Range:   base.ResolveRange(fmBodyStart, fmBodyEnd),
			Message: fmt.Sprintf("invalid YAML frontmatter: %s", err.Error()),
			Cause:   err,
		}
	}
	doc.Frontmatter.Keys = keys
	doc.fm = decoded
	return doc, nil
}

// collectFrontmatterKeys walks the top-level mapping of the
// frontmatter and returns (1) a key → file-level Range map and (2) a
// plain map[string]any of the decoded values. Non-mapping frontmatter
// (e.g. a YAML sequence) is rejected.
func collectFrontmatterKeys(file *ast.File, fmStart int, base *Base) (map[string]diag.Range, map[string]any, error) {
	if file == nil || len(file.Docs) == 0 {
		return map[string]diag.Range{}, map[string]any{}, nil
	}
	body := file.Docs[0].Body
	if body == nil {
		return map[string]diag.Range{}, map[string]any{}, nil
	}
	mapping, err := asMapping(body)
	if err != nil {
		return nil, nil, err
	}

	keys := make(map[string]diag.Range, len(mapping.Values))
	decoded := make(map[string]any, len(mapping.Values))
	for _, mv := range mapping.Values {
		keyNode, ok := mv.Key.(*ast.StringNode)
		if !ok {
			return nil, nil, fmt.Errorf("unsupported frontmatter key type %T", mv.Key)
		}
		tok := keyNode.GetToken()
		keyStart := fmStart + tok.Position.Offset - 1
		keyEnd := keyStart + len(tok.Value)
		keys[keyNode.Value] = base.ResolveRange(keyStart, keyEnd)

		v, err := yamlValue(mv.Value)
		if err != nil {
			return nil, nil, fmt.Errorf("key %q: %w", keyNode.Value, err)
		}
		decoded[keyNode.Value] = v
	}
	return keys, decoded, nil
}

// asMapping coerces the top-level frontmatter body into a
// *ast.MappingNode. goccy emits a bare *MappingValueNode when the
// document is a single key:value pair, so we wrap it into a
// single-element mapping for uniform downstream handling.
func asMapping(body ast.Node) (*ast.MappingNode, error) {
	if m, ok := body.(*ast.MappingNode); ok {
		return m, nil
	}
	if mv, ok := body.(*ast.MappingValueNode); ok {
		return &ast.MappingNode{Values: []*ast.MappingValueNode{mv}}, nil
	}
	return nil, errors.New("frontmatter must be a YAML mapping")
}

// yamlValue converts a goccy AST value node to a plain Go value. The
// shapes we care about are strings and string lists; other scalars
// fall back to their string representation so we do not lose
// information but also do not commit to richer typing than callers
// need.
func yamlValue(n ast.Node) (any, error) {
	if n == nil {
		return nil, nil //nolint:nilnil // YAML null is a legitimate value.
	}
	switch v := n.(type) {
	case *ast.StringNode:
		return v.Value, nil
	case *ast.IntegerNode:
		return v.Value, nil
	case *ast.FloatNode:
		return v.Value, nil
	case *ast.BoolNode:
		return v.Value, nil
	case *ast.NullNode:
		return nil, nil //nolint:nilnil // YAML null is a legitimate value.
	case *ast.SequenceNode:
		out := make([]any, 0, len(v.Values))
		for _, item := range v.Values {
			iv, err := yamlValue(item)
			if err != nil {
				return nil, err
			}
			out = append(out, iv)
		}
		return out, nil
	case *ast.MappingNode:
		out := make(map[string]any, len(v.Values))
		for _, mv := range v.Values {
			kn, ok := mv.Key.(*ast.StringNode)
			if !ok {
				return nil, fmt.Errorf("unsupported nested key type %T", mv.Key)
			}
			iv, err := yamlValue(mv.Value)
			if err != nil {
				return nil, err
			}
			out[kn.Value] = iv
		}
		return out, nil
	default:
		return n.String(), nil
	}
}

// asString returns the string value for key, or "" if the key is
// absent or is not a string.
func (d *markdownDoc) asString(key string) string {
	if s, ok := d.fm[key].(string); ok {
		return s
	}
	return ""
}

// asStringList returns the string-list value for key. It accepts both
// a YAML sequence of strings and a single string (coerced into a
// one-element slice). Non-string items fall back to "" but the list
// is still returned so rules can report a clear "item N is not a
// string" diagnostic; absence yields nil.
func (d *markdownDoc) asStringList(key string) []string {
	v, ok := d.fm[key]
	if !ok {
		return nil
	}
	switch vv := v.(type) {
	case string:
		return []string{vv}
	case []any:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			if s, ok := item.(string); ok {
				out = append(out, s)
				continue
			}
			out = append(out, "")
		}
		return out
	default:
		return nil
	}
}

// cleanYAMLError strips goccy's helpful-but-multi-line error decoration
// down to the first line, which is usually the cause. Reporters render
// ParseError.Message on a single line so preserving the full trace
// would produce noisy output; the original error is still reachable
// via Unwrap for tooling that wants it.
func cleanYAMLError(err error) string {
	msg := err.Error()
	if i := bytes.IndexByte([]byte(msg), '\n'); i >= 0 {
		msg = msg[:i]
	}
	return msg
}
