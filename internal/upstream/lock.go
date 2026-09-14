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
// Every field is derived from the response: a URL, a content hash, the
// server's own Last-Modified, and the release marker found in the page.
// Nothing is a clock reading, so re-running the tool against unchanged
// sources rewrites the file byte for byte.

// LockEntry is the committed state of one source.
//
// Fields are declared in alphabetical order of their json tag, for the
// same reason Digest's are: Go emits struct fields in declaration order.
type LockEntry struct {
	// LastModified is the server's Last-Modified header, verbatim. It is
	// advisory: not every source sends one, and the hash is what decides
	// whether the content changed.
	LastModified string `json:"last_modified,omitempty"`
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
// produced. A source the manifest has no successful record for is left
// out, which is how an optional probe that did not answer stays out of
// the committed file.
func NewLock(m Manifest, pages map[string][]byte) Lock {
	lock := make(Lock, len(m))

	for id, rec := range m {
		if rec.SHA256 == "" {
			continue
		}

		lock[id] = LockEntry{
			LastModified:  rec.LastModified,
			SHA256:        rec.SHA256,
			URL:           rec.URL,
			VersionMarker: VersionMarker(pages[id]),
		}
	}

	return lock
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
