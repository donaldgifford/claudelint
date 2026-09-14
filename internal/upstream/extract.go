package upstream

import (
	"errors"
	"fmt"
	"regexp"
)

// Reasons an extractor gives up. Each is wrapped in an *ExtractError
// that names the source, the digest section, and the anchor.
var (
	// ErrAnchorNotFound means the heading an extractor scopes to is no
	// longer on the page.
	ErrAnchorNotFound = errors.New("anchor heading not found")
	// ErrTableNotFound means the anchored section contains no GFM table.
	ErrTableNotFound = errors.New("no table in section")
	// ErrHeaders means the table's columns are not the ones expected.
	ErrHeaders = errors.New("unexpected table headers")
	// ErrEmptySection means the extractor ran but found nothing.
	ErrEmptySection = errors.New("section extracted no items")
	// ErrBelowFloor means the extracted count fell under the sanity
	// floor derived from the committed digest.
	ErrBelowFloor = errors.New("count below sanity floor")
	// ErrMissingSource means the source an extractor needs was not
	// fetched.
	ErrMissingSource = errors.New("source not fetched")
)

// ExtractError reports a structural failure in one extractor.
//
// Extraction fails hard rather than degrading: a digest missing one
// section would diff as a mass removal, which reads like upstream
// deleting a feature. Exiting with nothing written and a message naming
// the extractor is the honest outcome (DESIGN-0006 OQ11).
type ExtractError struct {
	// Source is the source id the extractor reads.
	Source string
	// Section is the digest path the extractor fills.
	Section string
	// Anchor is the heading pattern it scopes to, empty for
	// non-Markdown sources.
	Anchor string
	// Reason is one of the Err values above, possibly wrapped.
	Reason error
}

func (e *ExtractError) Error() string {
	if e.Anchor == "" {
		return fmt.Sprintf("extract %s from %s: %v", e.Section, e.Source, e.Reason)
	}

	return fmt.Sprintf("extract %s from %s (anchor %s): %v", e.Section, e.Source, e.Anchor, e.Reason)
}

// Unwrap exposes the reason so callers can use errors.Is.
func (e *ExtractError) Unwrap() error { return e.Reason }

// Extractor fills one digest section from one fetched source.
//
// The runner resolves the anchor, so every extractor reports a missing
// heading the same way and the section bytes it consumed can be written
// out as a golden snippet. An extractor with a nil anchor receives the
// whole page.
type Extractor interface {
	// Section is the dotted digest path this extractor fills. It keys
	// the sanity floor and labels errors.
	Section() string
	// Source is the source id from the table in source.go.
	Source() string
	// Anchor is the heading pattern to scope to, or nil to receive the
	// whole page.
	Anchor() *regexp.Regexp
	// Extract parses the scoped section and writes into out.
	Extract(section string, out *Digest) error
}

// Extractors returns every extractor in run order.
//
// This is an explicit slice rather than an init-time registry because,
// unlike the rule packages, every extractor lives in this one package.
// A registry would buy nothing and cost package-level mutable state and
// an implicit ordering.
func Extractors() []Extractor {
	return []Extractor{
		skillsFrontmatter{},
		skillsPortableFields{},
		skillsSubstitutions{},

		agentsFrontmatter{},
		agentsEnums{},

		pluginsManifestFields{},
		pluginsLocations{},
		pluginsSubstitutionFields{},

		marketplaceFields{},
		marketplaceOwnerFields{},
		marketplacePluginEntryFields{},
		marketplaceSources{},
		marketplaceReservedNames{},

		hooksEvents{},
		hooksTypes{},
		hooksHandlerFields{},
		hooksTimeoutDefaults{},

		mcpTransports{},
		mcpServerFields{},

		toolsBuiltin{},

		schemaStorePluginManifest{},
		schemaStoreMarketplaceSchema{},
		schemaStoreSettingsSchema{},

		portableSpec{},
		portableValidator{},
		portableAnthropic{},

		changelogLatest{},
	}
}

// ExtractOptions tunes a run.
type ExtractOptions struct {
	// Baseline is the committed digest, used to derive per-section
	// sanity floors. A nil baseline disables the floor check, which is
	// what bootstrapping the very first digest needs.
	Baseline *Digest
	// Snippets, when non-nil, is filled with the golden fixture content
	// for each extractor, keyed by its path under the snippets
	// directory. Extractors that share an anchor share one entry. This
	// is how the fixtures under testdata/snippets are produced.
	Snippets map[string]string
}

// Extract runs every extractor over the fetched pages and returns the
// assembled digest. It stops at the first failure and returns no
// digest.
func Extract(pages map[string][]byte, opts ExtractOptions) (*Digest, error) {
	d := NewDigest()

	for _, e := range Extractors() {
		section, err := sectionFor(e, pages)
		if err != nil {
			return nil, err
		}
		if opts.Snippets != nil {
			opts.Snippets[SnippetPath(e)] = Snippet(e, section)
		}
		if err := e.Extract(section, d); err != nil {
			return nil, asExtractError(e, err)
		}
	}

	d.Meta.DocsMaxMarker = maxDocsMarker(pages)
	d.Normalize()

	if err := checkFloors(d, opts.Baseline); err != nil {
		return nil, err
	}

	return d, nil
}

// sectionFor resolves the page and anchor for one extractor.
func sectionFor(e Extractor, pages map[string][]byte) (string, error) {
	page, ok := pages[e.Source()]
	if !ok {
		return "", asExtractError(e, ErrMissingSource)
	}

	anchor := e.Anchor()
	if anchor == nil {
		return string(page), nil
	}

	section, found := Section(page, anchor)
	if !found {
		return "", asExtractError(e, ErrAnchorNotFound)
	}

	return section, nil
}

// asExtractError wraps err with the extractor's identity, unless it
// already carries one.
func asExtractError(e Extractor, err error) error {
	var already *ExtractError
	if errors.As(err, &already) {
		return err
	}

	ex := &ExtractError{Source: e.Source(), Section: e.Section(), Reason: err}
	if a := e.Anchor(); a != nil {
		ex.Anchor = a.String()
	}

	return ex
}

// checkFloors rejects a digest whose sections shrank implausibly.
//
// A floor catches the failure mode a missing-anchor check cannot: an
// upstream edit that keeps the heading but guts the table, which would
// otherwise diff as upstream deleting most of a feature.
func checkFloors(head, baseline *Digest) error {
	if baseline == nil {
		return nil
	}

	baseTree, err := baseline.Tree()
	if err != nil {
		return err
	}
	headTree, err := head.Tree()
	if err != nil {
		return err
	}

	for _, e := range Extractors() {
		path := e.Section()

		want, ok := Count(baseTree, path)
		if !ok {
			continue
		}
		got, ok := Count(headTree, path)
		if !ok {
			continue
		}

		if limit := floor(want); got < limit {
			return asExtractError(e, fmt.Errorf("%w: %d rows, floor %d (committed %d)",
				ErrBelowFloor, got, limit, want))
		}
	}

	return nil
}

// floor is the minimum acceptable count for a section whose committed
// count is n: max(1, ceil(0.75*n)). A quarter of a section can vanish
// in one upstream edit without tripping it; more than that is a
// structural break, not a spec change.
func floor(n int) int {
	if n <= 0 {
		return 1
	}

	return (3*n + 3) / 4
}

// docsVersionMarker matches the release markers the documentation pages
// carry, such as "v2.1.268".
var docsVersionMarker = regexp.MustCompile(`\bv(\d+)\.(\d+)\.(\d+)\b`)

// VersionMarker returns the highest release marker on a page, or "".
func VersionMarker(page []byte) string {
	best := ""
	bestParts := [3]int{}

	for _, m := range docsVersionMarker.FindAllSubmatch(page, -1) {
		parts := [3]int{atoi(m[1]), atoi(m[2]), atoi(m[3])}
		if best == "" || parts[0] > bestParts[0] ||
			(parts[0] == bestParts[0] && parts[1] > bestParts[1]) ||
			(parts[0] == bestParts[0] && parts[1] == bestParts[1] && parts[2] > bestParts[2]) {
			best, bestParts = string(m[0]), parts
		}
	}

	return best
}

// maxDocsMarker returns the highest version marker across the
// documentation pages. It is metadata: a change here never counts as
// drift.
func maxDocsMarker(pages map[string][]byte) string {
	best := ""
	bestParts := [3]int{}

	for _, s := range Sources() {
		if s.Tier != TierAuthoritative {
			continue
		}
		page, ok := pages[s.ID]
		if !ok {
			continue
		}

		marker := VersionMarker(page)
		if marker == "" {
			continue
		}

		m := docsVersionMarker.FindStringSubmatch(marker)
		parts := [3]int{atoi([]byte(m[1])), atoi([]byte(m[2])), atoi([]byte(m[3]))}
		if best == "" || parts[0] > bestParts[0] ||
			(parts[0] == bestParts[0] && parts[1] > bestParts[1]) ||
			(parts[0] == bestParts[0] && parts[1] == bestParts[1] && parts[2] > bestParts[2]) {
			best, bestParts = marker, parts
		}
	}

	return best
}

// atoi parses a non-negative decimal run, returning 0 on anything else.
// The input always comes from a regexp capture of \d+, so a parse error
// is unreachable.
func atoi(b []byte) int {
	n := 0
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}

	return n
}
