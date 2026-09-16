// Package spec holds the committed upstream digest and the two facts
// about it that the linter itself needs.
//
// It exists to keep weight out of the release binary. The parent
// package fetches over HTTP, scans Markdown tables, and carries a cobra
// command tree, none of which belongs on the import path of a linter
// that only wants to print which documentation revision it was built
// against. Importing the parent for that one line cost the claudelint
// binary 1.3 MB, which is what this package avoids.
//
// Everything here is a leaf: the standard library and nothing else.
package spec

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
)

// DigestFile is the committed digest, relative to the repository root.
const DigestFile = "internal/upstream/spec/digest.json"

//go:embed digest.json
var digest []byte

// Version is the identity of the committed digest: which documentation
// revision it was extracted from, and a short hash of the bytes so two
// reports can be compared without diffing 22 KB of JSON.
type Version struct {
	// Marker is the highest version marker seen across the
	// documentation pages, with a leading "v".
	Marker string
	// Fingerprint is the first eight hex characters of the digest's
	// sha256, in the shape of the ruleset fingerprint beside it.
	Fingerprint string
}

// String renders the pair the way `claudelint version` prints it.
func (v Version) String() string { return v.Marker + " (" + v.Fingerprint + ")" }

// Bytes returns the committed digest exactly as it is compiled in.
func Bytes() []byte { return slices.Clone(digest) }

// Current reads the compiled-in digest's identity.
//
// It decodes only what it needs. `claudelint version` runs on every bug
// report, and it has no reason to pay for the whole digest tree to
// print one line.
func Current() (Version, error) {
	var head struct {
		Meta struct {
			DocsMaxMarker string `json:"docs_max_marker"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(digest, &head); err != nil {
		return Version{}, fmt.Errorf("embedded digest: %w", err)
	}

	sum := sha256.Sum256(digest)

	return Version{
		Marker:      head.Meta.DocsMaxMarker,
		Fingerprint: hex.EncodeToString(sum[:])[:8],
	}, nil
}
