package upstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
)

// LockFile is the committed record of what each source looked like when
// the digest was last regenerated.
const LockFile = "sources.lock.json"

// The lock answers a question the digest cannot: whether upstream
// changed at all.
//
// A digest only carries the facts the extractors read, so an upstream
// edit to prose, examples, or ordering leaves it identical. That is the
// point, and it is also confusing on its own: a reader who knows the
// documentation moved wants to see that the tool noticed. Recording the
// content hash of every source gives the report a "sources changed
// without affecting the digest" line and turns a silent no-op into an
// explicit one.
//
// Every field is derived from the content: a URL, a hash of the body,
// and the release marker found in the page. Nothing is a clock reading,
// so re-running the tool against unchanged sources rewrites the file
// byte for byte.
//
// Two things DESIGN-0006 put in the lock are deliberately absent,
// because measuring them showed they break that invariant:
//
// Last-Modified. The design records the header per source. On
// code.claude.com it is the time of the request, not of the content:
// two fetches five seconds apart report timestamps five seconds apart
// for identical bytes. Committing it would make every sync a diff. The
// header is still recorded in the work-directory manifest, where a
// clock reading is diagnostic rather than committed.
//
// Optional probes. The two code.claude.com/schemas URLs do not exist
// yet and the site answers them with its product page, whose body
// carries a per-request nonce. A probe answers a yes-or-no question
// about a URL, and its body is not a specification until someone
// writes an extractor for it, so it stays out of the committed
// record.

// LockEntry is the committed state of one source.
//
// Fields are declared in alphabetical order of their json tag, for the
// same reason Digest's are: Go emits struct fields in declaration order.
type LockEntry struct {
	// SHA256 is the hex digest of the response body.
	SHA256 string `json:"sha256"`
	// URL is the final URL after redirects.
	URL string `json:"url"`
	// VersionMarker is the highest release marker found in the page, for
	// the sources that carry one.
	VersionMarker string `json:"version_marker,omitempty"`
}

// Lock maps a source id to its committed state. It is a map so the
// encoder sorts the keys.
type Lock map[string]LockEntry

// NewLock builds a lock from a fetch manifest and the pages it
// produced. A source with no successful record, and any optional probe,
// is left out.
func NewLock(m Manifest, pages map[string][]byte) Lock {
	lock := make(Lock, len(m))

	for id, rec := range m {
		if rec.SHA256 == "" || isOptional(id) {
			continue
		}

		lock[id] = LockEntry{
			SHA256:        rec.SHA256,
			URL:           rec.URL,
			VersionMarker: VersionMarker(pages[id]),
		}
	}

	return lock
}

// isOptional reports whether a source id belongs to an optional probe.
// An id the table does not know is treated as required, so a manifest
// entry is never dropped by accident.
func isOptional(id string) bool {
	s, ok := SourceByID(id)

	return ok && s.Optional
}

// Encode returns the canonical lock bytes: key-sorted, two-space
// indent, LF, one trailing newline.
func (l Lock) Encode() ([]byte, error) {
	var buf bytes.Buffer
	if err := encodeJSON(&buf, l); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecodeLock parses lock bytes.
func DecodeLock(data []byte) (Lock, error) {
	var l Lock
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("decode lock: %w", err)
	}

	return l, nil
}

// LoadLock reads a lock file. A missing file is not an error: the first
// run has nothing to compare against and gets an empty lock.
func LoadLock(path string) (Lock, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Lock{}, nil
		}

		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return DecodeLock(raw)
}

// ChangedSources returns the ids whose content hash differs between two
// locks, including sources present in only one of them. The result is
// sorted.
func ChangedSources(before, after Lock) []string {
	var changed []string

	for id, entry := range after {
		if prev, ok := before[id]; !ok || prev.SHA256 != entry.SHA256 {
			changed = append(changed, id)
		}
	}

	for id := range before {
		if _, ok := after[id]; !ok {
			changed = append(changed, id)
		}
	}

	slices.Sort(changed)

	return slices.Compact(changed)
}

// SourceIDs returns the ids in a lock, sorted.
func (l Lock) SourceIDs() []string {
	return slices.Sorted(maps.Keys(l))
}
