package upstream_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

// A guardrail that cannot fail is decoration. These tests drive the
// same comparison the real one uses with synthetic data, so the two
// failure modes that matter are proved rather than assumed: code that
// falls behind the documentation, and an acknowledgement that outlives
// the difference it excused.

// TestGuardrailDetectsAMissingItem is the first Phase 2 success
// criterion: dropping a hook event from the Go set produces a message
// naming the identifier, the digest path, and the event.
func TestGuardrailDetectsAMissingItem(t *testing.T) {
	t.Parallel()

	g := &guard{ack: upstream.Acknowledged{}}

	msg := g.equalSetsDrift(
		"hooks.events", "artifact.KnownHookEvents",
		[]string{"PreToolUse", "PreModelSwitch"},
		[]string{"PreToolUse"},
	)

	if msg == "" {
		t.Fatal("equalSetsDrift() = \"\", want a failure message")
	}
	for _, want := range []string{"artifact.KnownHookEvents", "hooks.events", "PreModelSwitch", "acknowledged.json"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message does not mention %q:\n%s", want, msg)
		}
	}
}

// TestGuardrailDetectsAnUndocumentedItem covers the other direction: a
// tool in the Go set that upstream no longer lists.
func TestGuardrailDetectsAnUndocumentedItem(t *testing.T) {
	t.Parallel()

	g := &guard{ack: upstream.Acknowledged{}}

	msg := g.equalSetsDrift(
		"tools.builtin", "artifact.KnownTools",
		[]string{"Read"},
		[]string{"Read", "MultiEdit"},
	)

	if !strings.Contains(msg, "MultiEdit") || !strings.Contains(msg, "not documented upstream") {
		t.Errorf("message does not report the undocumented tool:\n%s", msg)
	}
}

// TestAcknowledgementSuppressesOneItemOnly is the property that makes
// the file safe: excusing one item says nothing about its neighbours.
func TestAcknowledgementSuppressesOneItemOnly(t *testing.T) {
	t.Parallel()

	g := &guard{ack: upstream.Acknowledged{
		"tools.builtin": {"Task": "renamed in v2.1.63"},
	}}

	if msg := g.equalSetsDrift("tools.builtin", "artifact.KnownTools",
		[]string{"Read"}, []string{"Read", "Task"}); msg != "" {
		t.Errorf("an acknowledged item still failed:\n%s", msg)
	}

	msg := g.equalSetsDrift("tools.builtin", "artifact.KnownTools",
		[]string{"Read"}, []string{"Read", "Task", "MultiEdit"})
	if !strings.Contains(msg, "MultiEdit") {
		t.Errorf("message does not report the unacknowledged item:\n%s", msg)
	}
	if strings.Contains(msg, "Task") {
		t.Errorf("message reports the acknowledged item:\n%s", msg)
	}
}

// TestStaleAcknowledgementIsReported is the second half of the first
// success criterion: an acknowledgement for something that is not a
// difference fails, so the file self-cleans.
func TestStaleAcknowledgementIsReported(t *testing.T) {
	t.Parallel()

	g := &guard{ack: upstream.Acknowledged{
		"tools.builtin": {"Frobnicate": "invented"},
	}}

	// A clean comparison uses no acknowledgement.
	if msg := g.equalSetsDrift("tools.builtin", "artifact.KnownTools",
		[]string{"Read"}, []string{"Read"}); msg != "" {
		t.Fatalf("equalSetsDrift() on identical sets = %q, want none", msg)
	}

	stale := g.staleAcknowledgements()
	if len(stale) != 1 || !strings.Contains(stale[0], "Frobnicate") {
		t.Errorf("staleAcknowledgements() = %v, want the Frobnicate entry", stale)
	}
}

// TestCoversAllowsExtraParsedKeys pins the one-way relation: the
// parsers read when_to_use and context, which the published table does
// not list, and that is not drift.
func TestCoversAllowsExtraParsedKeys(t *testing.T) {
	t.Parallel()

	g := &guard{ack: upstream.Acknowledged{}}

	if msg := g.coversDrift("skills.frontmatter", "artifact.SkillFrontmatterKeys",
		[]string{"name"}, []string{"name", "when_to_use"}); msg != "" {
		t.Errorf("an extra parsed key failed the covers relation:\n%s", msg)
	}

	if msg := g.coversDrift("skills.frontmatter", "artifact.SkillFrontmatterKeys",
		[]string{"name", "paths"}, []string{"name"}); msg == "" {
		t.Error("a documented key nothing parses passed the covers relation")
	}
}

func TestAcknowledgedValidate(t *testing.T) {
	t.Parallel()

	digest, _, err := upstream.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error = %v", err)
	}

	tests := []struct {
		name string
		ack  upstream.Acknowledged
		want error
	}{
		{
			name: "valid",
			ack:  upstream.Acknowledged{"tools.builtin": {"Task": "renamed in v2.1.63"}},
		},
		{
			name: "unknown path",
			ack:  upstream.Acknowledged{"tools.imaginary": {"Task": "renamed"}},
			want: upstream.ErrUnknownPath,
		},
		{
			name: "empty reason",
			ack:  upstream.Acknowledged{"tools.builtin": {"Task": "   "}},
			want: upstream.ErrEmptyReason,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.ack.Validate(digest)
			switch {
			case tc.want == nil && err != nil:
				t.Errorf("Validate() error = %v, want nil", err)
			case tc.want != nil && err == nil:
				t.Errorf("Validate() error = nil, want %v", tc.want)
			case tc.want != nil && !strings.Contains(err.Error(), tc.want.Error()):
				t.Errorf("Validate() error = %v, want it to wrap %v", err, tc.want)
			}
		})
	}
}

// TestEmbeddedDigestMatchesTheFile keeps the compiled-in copy honest.
// A release prints a hash of the embedded bytes; if those could differ
// from the committed file, the hash would name a digest nobody can
// read.
func TestEmbeddedDigestMatchesTheFile(t *testing.T) {
	t.Parallel()

	onDisk := readFile(t, filepath.Join("spec", "digest.json"))

	if !bytes.Equal(upstream.EmbeddedDigestBytes(), onDisk) {
		t.Error("the embedded digest differs from internal/upstream/digest.json")
	}
}

// TestAcknowledgedFileIsCanonical keeps the committed file formatted
// the way the digest and the lock are, so a hand edit does not produce
// a whole-file diff on the next tool-written change.
func TestAcknowledgedFileIsCanonical(t *testing.T) {
	t.Parallel()

	raw := readFile(t, upstream.AcknowledgedFile)

	ack, err := upstream.DecodeAcknowledged(raw)
	if err != nil {
		t.Fatalf("DecodeAcknowledged() error = %v", err)
	}

	got, err := ack.Encode()
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if !bytes.Equal(got, raw) {
		t.Errorf("%s is not canonically formatted; rewrite it as:\n%s",
			upstream.AcknowledgedFile, got)
	}
}
