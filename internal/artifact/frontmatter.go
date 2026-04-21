package artifact

import "github.com/donaldgifford/claudelint/internal/diag"

// Frontmatter is the positional metadata every Markdown artifact
// carries about its YAML frontmatter block. Keys is keyed by
// frontmatter field name and maps to the Range of that key's line —
// parsers populate it as they walk the YAML so rules can point
// diagnostics at exact fields.
//
// A missing key is simply absent from Keys; it does not have a zero
// Range entry. Callers should use Range.IsZero() on Block to decide
// whether the artifact had frontmatter at all.
type Frontmatter struct {
	// Block is the Range of the entire --- ... --- fenced block.
	// Zero Range means the artifact has no frontmatter.
	Block diag.Range

	// Keys maps each parsed field name to the Range of its key token.
	Keys map[string]diag.Range
}

// KeyRange returns the Range of key, or a zero Range if the key was
// not present. Rules should prefer this over touching Keys directly so
// a missing key lands on a harmless zero Range rather than a lookup
// panic.
func (f Frontmatter) KeyRange(key string) diag.Range {
	if f.Keys == nil {
		return diag.Range{}
	}
	return f.Keys[key]
}
