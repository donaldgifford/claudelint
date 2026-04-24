package artifact

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/buger/jsonparser"

	"github.com/donaldgifford/claudelint/internal/diag"
)

// ParseMarketplace parses a .claude-plugin/marketplace.json manifest.
// It mirrors ParsePlugin's shape: syntactic errors are reported as a
// *ParseError pointing at the offending bytes; missing optional fields
// are tolerated and left zero on the returned Marketplace.
//
// plugins[].source values are resolved to repo-relative local paths
// relative to the marketplace manifest's parent directory. External
// sources (git URLs: github:..., https://..., git@...) are preserved
// in Source but yield Resolved == "". The resolution is a syntactic
// convenience for discovery; it does not stat the filesystem.
func ParseMarketplace(filePath string, src []byte) (*Marketplace, *ParseError) {
	base := NewBase(filePath, src)

	if !isJSONFile(filePath) {
		return nil, &ParseError{
			Path:    filePath,
			Range:   base.ResolveRange(0, len(src)),
			Message: "marketplace manifests must be JSON (.claude-plugin/marketplace.json)",
		}
	}
	if err := validateJSON(src); err != nil {
		return nil, &ParseError{
			Path:    filePath,
			Range:   base.ResolveRange(0, len(src)),
			Message: fmt.Sprintf("invalid JSON: %s", err.Error()),
			Cause:   err,
		}
	}

	m := &Marketplace{Base: base}
	m.Name, m.NameRange = stringField(src, &base, "name")
	m.Version, m.VersionRange = stringField(src, &base, "version")
	m.Author, m.AuthorRange = stringField(src, &base, "author")
	m.Plugins = parseMarketplacePlugins(src, &base, marketplaceRoot(filePath))

	return m, nil
}

// parseMarketplacePlugins iterates the plugins[] array and resolves
// each entry's source. Non-object entries are skipped silently; a
// missing plugins field yields a nil slice, not an error — rules
// decide whether that is meaningful.
func parseMarketplacePlugins(src []byte, base *Base, root string) []MarketplacePlugin {
	pluginsRaw, dt, pluginsEndAbs, err := jsonparser.Get(src, "plugins")
	if err != nil || dt != jsonparser.Array {
		return nil
	}
	// jsonparser's returned offset is the end of the value. The array
	// bytes run [pluginsStartAbs, pluginsEndAbs); compute the start so
	// we can translate item-relative offsets back to absolute.
	pluginsStartAbs := pluginsEndAbs - len(pluginsRaw)

	var out []MarketplacePlugin
	_, aerr := jsonparser.ArrayEach(pluginsRaw, func(item []byte, itemDT jsonparser.ValueType, itemOff int, _ error) {
		if itemDT != jsonparser.Object {
			return
		}
		itemAbs := pluginsStartAbs + itemOff

		mp := MarketplacePlugin{}
		mp.Name, mp.NameRange = stringFieldAt(item, itemAbs, base, "name")
		mp.Source, mp.SourceRange = stringFieldAt(item, itemAbs, base, "source")
		mp.Resolved = resolveMarketplaceSource(root, mp.Source)
		out = append(out, mp)
	})
	if aerr != nil && !errors.Is(aerr, jsonparser.KeyPathNotFoundError) {
		return out
	}
	return out
}

// stringFieldAt is stringField's analogue for a string value nested
// inside an array item. itemAbs is the item's absolute start offset
// in the file; the function converts item-relative offsets back to
// absolute so ranges point at the real file.
//
// jsonparser returns the offset past the value — for a string that is
// one byte past the closing quote. The full quoted span therefore runs
// from (endAbs - len(value) - 2) (opening quote) to endAbs (exclusive
// end past the closing quote).
func stringFieldAt(item []byte, itemAbs int, base *Base, key string) (string, diag.Range) {
	value, dt, endOff, err := jsonparser.Get(item, key)
	if err != nil || dt != jsonparser.String {
		return "", diag.Range{}
	}
	endAbs := itemAbs + endOff
	startAbs := max(endAbs-len(value)-2, 0)
	return string(value), base.ResolveRange(startAbs, endAbs)
}

// marketplaceRoot returns the repo-relative directory that a
// marketplace's `source: "./"` resolves to: the parent of the
// .claude-plugin/ directory. For a repo-root manifest the root is
// "." (an empty string would be ambiguous with a missing path).
func marketplaceRoot(filePath string) string {
	p := path.Clean(filePath)
	// Strip the known suffix. Classification has already accepted the
	// file, so this suffix is guaranteed to be present.
	const suffix = ".claude-plugin/marketplace.json"
	if p == suffix {
		return "."
	}
	return strings.TrimSuffix(p, "/"+suffix)
}

// resolveMarketplaceSource converts a plugins[].source value into a
// repo-relative local path, or "" for an external (git URL) entry.
//
// External heuristics (each reliably detectable from the string
// alone, no filesystem access):
//
//   - starts with "github:"       — short form per the marketplaces spec
//   - contains "://"              — covers https://, http://, git://, ssh://
//   - starts with "git@"          — scp-style ssh URL
func resolveMarketplaceSource(root, source string) string {
	if source == "" {
		return ""
	}
	if strings.HasPrefix(source, "github:") ||
		strings.Contains(source, "://") ||
		strings.HasPrefix(source, "git@") {
		return ""
	}
	// Local source: join with the marketplace root and clean.
	// source may be "./" or "./plugins/foo" or "plugins/foo".
	joined := path.Join(root, source)
	return path.Clean(joined)
}
