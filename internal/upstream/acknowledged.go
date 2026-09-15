package upstream

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/donaldgifford/claudelint/internal/upstream/spec"
)

// AcknowledgedFile is the committed record of deliberate deviations.
const AcknowledgedFile = "acknowledged.json"

// An acknowledgement is how a deliberate deviation stays deliberate.
//
// The guardrail compares what the documentation says with what the
// rules encode. Some of those differences are choices, not bugs: a tool
// the runtime still accepts after the docs dropped it, a documented
// field no rule consumes yet. Without a record, each one is either a
// permanently failing test or a silently skipped comparison, and both
// rot.
//
// An acknowledgement names the digest path, the item, and the reason.
// It suppresses the failure for that one item and nothing else, and the
// reason has to name the rule or the phase that would resolve it, so a
// reader can tell a decision from a shrug.
//
// The file self-cleans: the guardrail fails an acknowledgement whose
// item is no longer a difference, which is what stops the list becoming
// a graveyard.

//go:embed acknowledged.json
var embeddedAcknowledged []byte

// Acknowledged maps a digest path to the items excused under it, each
// with its reason.
type Acknowledged map[string]map[string]string

// ErrUnknownPath means an acknowledgement names a digest path that does
// not exist, which is almost always a typo.
var ErrUnknownPath = errors.New("unknown digest path")

// ErrEmptyReason means an acknowledgement carries no reason. A blank
// reason is indistinguishable from a suppression, which is the thing
// this file exists to prevent.
var ErrEmptyReason = errors.New("acknowledgement has no reason")

// LoadEmbedded returns the digest and acknowledgements compiled into
// the binary, validated against each other. No filesystem is touched,
// so the guardrail test runs anywhere.
func LoadEmbedded() (*Digest, Acknowledged, error) {
	digest, err := Decode(spec.Bytes())
	if err != nil {
		return nil, nil, fmt.Errorf("embedded digest: %w", err)
	}

	ack, err := DecodeAcknowledged(embeddedAcknowledged)
	if err != nil {
		return nil, nil, err
	}

	if err := ack.Validate(digest); err != nil {
		return nil, nil, err
	}

	return digest, ack, nil
}

// EmbeddedDigestBytes returns the committed digest exactly as it is
// compiled in. A release can print a hash of it, and a test can compare
// it with the file on disk.
func EmbeddedDigestBytes() []byte { return spec.Bytes() }

// DecodeAcknowledged parses an acknowledgement file.
func DecodeAcknowledged(data []byte) (Acknowledged, error) {
	var a Acknowledged
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("decode %s: %w", AcknowledgedFile, err)
	}

	return a, nil
}

// Validate rejects an acknowledgement that names a digest path which
// does not exist, or that carries no reason.
func (a Acknowledged) Validate(d *Digest) error {
	tree, err := d.Tree()
	if err != nil {
		return err
	}

	for _, path := range a.Paths() {
		if _, ok := At(tree, path); !ok {
			return fmt.Errorf("%s: %w: %q", AcknowledgedFile, ErrUnknownPath, path)
		}

		for _, item := range slices.Sorted(maps.Keys(a[path])) {
			if strings.TrimSpace(a[path][item]) == "" {
				return fmt.Errorf("%s: %w: %s / %s", AcknowledgedFile, ErrEmptyReason, path, item)
			}
		}
	}

	return nil
}

// Paths returns the acknowledged digest paths, sorted.
func (a Acknowledged) Paths() []string { return slices.Sorted(maps.Keys(a)) }

// Items returns the items acknowledged under one path, sorted.
func (a Acknowledged) Items(path string) []string {
	return slices.Sorted(maps.Keys(a[path]))
}

// Reason returns the reason recorded for one item.
func (a Acknowledged) Reason(path, item string) (string, bool) {
	items, ok := a[path]
	if !ok {
		return "", false
	}
	reason, ok := items[item]

	return reason, ok
}

// Has reports whether one item is acknowledged under a path.
func (a Acknowledged) Has(path, item string) bool {
	_, ok := a.Reason(path, item)

	return ok
}

// Encode returns the canonical acknowledgement bytes, so the committed
// file is formatted the same way the digest and the lock are.
func (a Acknowledged) Encode() ([]byte, error) {
	var b strings.Builder
	if err := encodeJSON(&b, a); err != nil {
		return nil, err
	}

	return []byte(b.String()), nil
}
