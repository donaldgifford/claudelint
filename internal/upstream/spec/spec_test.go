package spec_test

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream/spec"
)

// This package exists to stay small. These tests are the two claims
// that makes worth anything: the compiled-in bytes are the committed
// file, and the version line the binary prints is derived from them
// rather than guessed.

func TestBytesMatchTheCommittedFile(t *testing.T) {
	t.Parallel()

	onDisk, err := os.ReadFile("digest.json")
	if err != nil {
		t.Fatalf("read digest.json: %v", err)
	}

	if !bytes.Equal(spec.Bytes(), onDisk) {
		t.Error("the embedded digest differs from digest.json")
	}
}

// TestBytesReturnsACopy keeps a caller from editing what every other
// caller reads.
func TestBytesReturnsACopy(t *testing.T) {
	t.Parallel()

	first := spec.Bytes()
	if len(first) == 0 {
		t.Fatal("Bytes() is empty")
	}
	first[0] = 'X'

	if spec.Bytes()[0] == 'X' {
		t.Error("Bytes() hands out the package's own slice")
	}
}

func TestCurrent(t *testing.T) {
	t.Parallel()

	got, err := spec.Current()
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if !strings.HasPrefix(got.Marker, "v2.") {
		t.Errorf("Marker = %q, want a documentation version marker", got.Marker)
	}
	if len(got.Fingerprint) != 8 {
		t.Errorf("Fingerprint = %q, want eight characters", got.Fingerprint)
	}
	if _, err := hex.DecodeString(got.Fingerprint); err != nil {
		t.Errorf("Fingerprint = %q, want hex: %v", got.Fingerprint, err)
	}

	if want := got.Marker + " (" + got.Fingerprint + ")"; got.String() != want {
		t.Errorf("String() = %q, want %q", got.String(), want)
	}
}

// TestDigestFilePointsAtThisPackage keeps the exported path honest for
// callers that resolve it from the repository root.
func TestDigestFilePointsAtThisPackage(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat(filepath.Join("..", "..", "..", spec.DigestFile)); err != nil {
		t.Errorf("DigestFile = %q: %v", spec.DigestFile, err)
	}
}
