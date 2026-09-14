package upstream_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

func TestNewLockFromManifest(t *testing.T) {
	t.Parallel()

	m := upstream.Manifest{
		"docs.hooks": {
			Bytes:        12,
			ETag:         `W/"abc"`,
			LastModified: "Fri, 11 Sep 2026 13:04:33 GMT",
			SHA256:       "3f1c",
			Status:       200,
			URL:          "https://code.claude.com/docs/en/hooks.md",
		},
	}
	pages := map[string][]byte{"docs.hooks": []byte("released in v2.1.265\n")}

	lock := upstream.NewLock(m, pages)

	entry, ok := lock["docs.hooks"]
	if !ok {
		t.Fatalf("NewLock() = %v, want an entry for docs.hooks", lock)
	}
	if entry.SHA256 != "3f1c" {
		t.Errorf("sha256 = %q, want %q", entry.SHA256, "3f1c")
	}
	if entry.VersionMarker != "v2.1.265" {
		t.Errorf("version_marker = %q, want %q", entry.VersionMarker, "v2.1.265")
	}
	if entry.LastModified != "Fri, 11 Sep 2026 13:04:33 GMT" {
		t.Errorf("last_modified = %q, want the response header", entry.LastModified)
	}
}

// TestNewLockSkipsUnfetchedSources covers the optional-probe case: a
// manifest entry with no hash records a failure, and a failure has no
// place in a committed lock.
func TestNewLockSkipsUnfetchedSources(t *testing.T) {
	t.Parallel()

	m := upstream.Manifest{
		"probe.plugin_schema": {Status: 404, URL: "https://example.test/missing.json"},
	}

	if lock := upstream.NewLock(m, nil); len(lock) != 0 {
		t.Fatalf("NewLock() = %v, want no entries", lock)
	}
}

// TestLockEncodeIsCanonical pins the properties the committed file
// depends on: sorted keys, two-space indent, and a trailing newline, so
// a run over unchanged sources rewrites identical bytes.
func TestLockEncodeIsCanonical(t *testing.T) {
	t.Parallel()

	lock := upstream.Lock{
		"docs.tools": {SHA256: "bbb", URL: "https://example.test/tools.md"},
		"docs.hooks": {SHA256: "aaa", URL: "https://example.test/hooks.md"},
	}

	got, err := lock.Encode()
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if !bytes.HasSuffix(got, []byte("}\n")) {
		t.Errorf("Encode() does not end with a newline: %q", got)
	}
	if bytes.Contains(got, []byte("\r")) {
		t.Error("Encode() emitted a carriage return")
	}
	if i, j := bytes.Index(got, []byte("docs.hooks")), bytes.Index(got, []byte("docs.tools")); i > j {
		t.Errorf("Encode() keys are not sorted:\n%s", got)
	}
	if !bytes.Contains(got, []byte("\n  \"docs.hooks\"")) {
		t.Errorf("Encode() is not two-space indented:\n%s", got)
	}

	again, err := lock.Encode()
	if err != nil {
		t.Fatalf("Encode() second call error = %v", err)
	}
	if !bytes.Equal(got, again) {
		t.Error("Encode() is not deterministic")
	}
}

// TestLockOmitsEmptyOptionalFields keeps the committed file free of keys
// that carry nothing, so a source that gains a Last-Modified header
// shows up as an addition rather than a change from "".
func TestLockOmitsEmptyOptionalFields(t *testing.T) {
	t.Parallel()

	lock := upstream.Lock{"docs.hooks": {SHA256: "aaa", URL: "https://example.test/hooks.md"}}

	got, err := lock.Encode()
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	for _, key := range []string{"last_modified", "version_marker"} {
		if bytes.Contains(got, []byte(key)) {
			t.Errorf("Encode() emitted empty %s:\n%s", key, got)
		}
	}
}

func TestLockRoundTrip(t *testing.T) {
	t.Parallel()

	lock := upstream.Lock{
		"docs.hooks": {
			LastModified:  "Fri, 11 Sep 2026 13:04:33 GMT",
			SHA256:        "aaa",
			URL:           "https://example.test/hooks.md",
			VersionMarker: "v2.1.265",
		},
	}

	raw, err := lock.Encode()
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	back, err := upstream.DecodeLock(raw)
	if err != nil {
		t.Fatalf("DecodeLock() error = %v", err)
	}
	if back["docs.hooks"] != lock["docs.hooks"] {
		t.Errorf("round trip = %+v, want %+v", back["docs.hooks"], lock["docs.hooks"])
	}
}

func TestDecodeLockRejectsGarbage(t *testing.T) {
	t.Parallel()

	if _, err := upstream.DecodeLock([]byte("not json")); err == nil {
		t.Fatal("DecodeLock() error = nil, want a decode error")
	}
}

func TestLoadLock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, upstream.LockFile)

	got, err := upstream.LoadLock(path)
	if err != nil {
		t.Fatalf("LoadLock() on a missing file error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Errorf("LoadLock() on a missing file = %v, want empty", got)
	}

	want := upstream.Lock{"docs.hooks": {SHA256: "aaa", URL: "https://example.test/hooks.md"}}
	raw, err := want.Encode()
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	got, err = upstream.LoadLock(path)
	if err != nil {
		t.Fatalf("LoadLock() error = %v", err)
	}
	if got["docs.hooks"].SHA256 != "aaa" {
		t.Errorf("LoadLock() = %+v, want the written entry", got)
	}
}

func TestChangedSources(t *testing.T) {
	t.Parallel()

	before := upstream.Lock{
		"docs.hooks": {SHA256: "aaa"},
		"docs.tools": {SHA256: "bbb"},
		"docs.mcp":   {SHA256: "ccc"},
	}
	after := upstream.Lock{
		"docs.hooks":  {SHA256: "aaa"},
		"docs.tools":  {SHA256: "changed"},
		"docs.skills": {SHA256: "ddd"},
	}

	want := []string{"docs.mcp", "docs.skills", "docs.tools"}
	if got := upstream.ChangedSources(before, after); !slices.Equal(got, want) {
		t.Errorf("ChangedSources() = %v, want %v", got, want)
	}

	if got := upstream.ChangedSources(before, before); len(got) != 0 {
		t.Errorf("ChangedSources() on an unchanged lock = %v, want none", got)
	}
}

func TestLockSourceIDs(t *testing.T) {
	t.Parallel()

	lock := upstream.Lock{"docs.tools": {}, "docs.hooks": {}}

	want := []string{"docs.hooks", "docs.tools"}
	if got := lock.SourceIDs(); !slices.Equal(got, want) {
		t.Errorf("SourceIDs() = %v, want %v", got, want)
	}
}

// TestLockCarriesNoTimestamp is the property that makes the file safe to
// commit: nothing in it comes from a clock, so a no-op run produces no
// diff.
func TestLockCarriesNoTimestamp(t *testing.T) {
	t.Parallel()

	lock := upstream.NewLock(upstream.Manifest{
		"docs.hooks": {SHA256: "aaa", URL: "https://example.test/hooks.md"},
	}, nil)

	got, err := lock.Encode()
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	for _, banned := range []string{"fetched_at", "generated", "timestamp", "updated_at"} {
		if strings.Contains(string(got), banned) {
			t.Errorf("Encode() emitted %q:\n%s", banned, got)
		}
	}
}
