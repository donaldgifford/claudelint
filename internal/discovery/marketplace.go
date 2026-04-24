package discovery

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// MarketplaceHints summarizes a discovered marketplace manifest.
// Discovery emits one MarketplaceHints when it sees
// .claude-plugin/marketplace.json at the walk root; engine-level
// callers (or future on-disk validation rules) can use it to enumerate
// declared plugin roots without re-parsing the manifest.
//
// External plugin entries (git URLs) are excluded from PluginRoots so
// the slice contains only locally-resolvable paths.
type MarketplaceHints struct {
	// ManifestPath is the repo-relative path of the manifest. Empty
	// when no manifest was found.
	ManifestPath string

	// PluginRoots is the deduplicated set of repo-relative directories
	// declared in the manifest's plugins[] array, slash-separated.
	// External entries (Resolved == "") are omitted.
	PluginRoots []string
}

// LoadMarketplaceHints reads <absRoot>/.claude-plugin/marketplace.json
// if present and returns the parsed hints. A missing manifest is not
// an error — callers receive a zero-value MarketplaceHints and (nil)
// for err. A malformed manifest is also tolerated: the hint is empty
// and the engine will surface the parse error through normal rule
// dispatch when the file gets re-parsed as an artifact.
//
// Reads are intentionally cheap: this helper does not cache, but
// marketplace manifests are tiny and discovery only calls it once
// per Walker invocation.
func LoadMarketplaceHints(absRoot string) (MarketplaceHints, error) {
	const rel = ".claude-plugin/marketplace.json"
	abs := filepath.Join(absRoot, rel)
	src, err := os.ReadFile(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return MarketplaceHints{}, nil
		}
		return MarketplaceHints{}, err
	}
	m, perr := artifact.ParseMarketplace(rel, src)
	if perr != nil {
		// Tolerate parse errors here — rules will report them with full
		// context when the file is parsed as an artifact. Returning an
		// empty hint keeps discovery quiet; surfacing the same error
		// twice would double-report.
		return MarketplaceHints{ManifestPath: rel}, nil //nolint:nilerr // reported downstream
	}

	roots := make([]string, 0, len(m.Plugins))
	seen := make(map[string]struct{}, len(m.Plugins))
	for i := range m.Plugins {
		p := &m.Plugins[i]
		if p.Resolved == "" {
			continue
		}
		clean := strings.TrimPrefix(p.Resolved, "./")
		if _, dup := seen[clean]; dup {
			continue
		}
		seen[clean] = struct{}{}
		roots = append(roots, clean)
	}
	return MarketplaceHints{ManifestPath: rel, PluginRoots: roots}, nil
}
