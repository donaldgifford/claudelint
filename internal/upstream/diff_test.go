package upstream_test

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

var updateGolden = flag.Bool("update", false, "rewrite the diff golden files")

const diffGoldenDir = "testdata/diff"

// diffCase is one before/after pair and the report it should produce.
type diffCase struct {
	// name keys the golden files.
	name string
	// mutate turns the baseline digest into the "after" side.
	mutate func(*upstream.Digest)
	// sources is the lock diff the caller would attach.
	sources []string
	// drift is whether the case should exit 1.
	drift bool
}

// baselineDigest is a small hand-built digest. It is deliberately not
// the fixture digest: a golden that changes whenever upstream does
// tests the fixtures, not the diff.
func baselineDigest() *upstream.Digest {
	d := upstream.NewDigest()

	d.Meta = upstream.Meta{ClaudeCodeLatest: "2.1.270", DocsMaxMarker: "v2.1.269"}
	d.Hooks.Events = []string{"PreToolUse", "PostToolUse", "SessionStart"}
	d.Hooks.Types = []string{"command", "http"}
	d.Hooks.TimeoutDefaults.ByType = map[string]int{"command": 600, "prompt": 30}
	d.Tools.Builtin = []string{"Bash", "Edit", "Read"}
	d.Skills.Frontmatter = []upstream.Field{
		{Name: "description", Required: upstream.RequiredYes},
		{Name: "name", Required: upstream.RequiredYes},
	}
	d.Plugins.ManifestFields = []upstream.Field{
		{Name: "hooks", Group: "component", Type: "string"},
		{Name: "hooks", Group: "metadata", Type: "boolean"},
	}
	d.SchemaStore.PluginManifest.HookEvents = []string{"PreToolUse"}
	d.Normalize()

	return d
}

func diffCases() []diffCase {
	return []diffCase{
		{
			name:    "noop",
			mutate:  func(*upstream.Digest) {},
			sources: []string{"docs.hooks", "docs.tools"},
		},
		{
			name: "added",
			mutate: func(d *upstream.Digest) {
				d.Hooks.Events = append(d.Hooks.Events, "PreModelSwitch", "PostModelSwitch")
				d.Tools.Builtin = append(d.Tools.Builtin, "Task")
			},
			drift: true,
		},
		{
			name: "removed",
			mutate: func(d *upstream.Digest) {
				d.Hooks.Events = []string{"PreToolUse", "PostToolUse"}
				d.Hooks.TimeoutDefaults.ByType = map[string]int{"command": 600}
			},
			drift: true,
		},
		{
			name: "changed",
			mutate: func(d *upstream.Digest) {
				d.Skills.Frontmatter[0].Required = upstream.RequiredRecommended
				d.Hooks.TimeoutDefaults.ByType["command"] = 300
				d.Plugins.ManifestFields[1].Type = "string"
			},
			drift: true,
		},
		{
			name: "informational",
			mutate: func(d *upstream.Digest) {
				d.Meta.ClaudeCodeLatest = "2.1.271"
				d.SchemaStore.PluginManifest.HookEvents = []string{"PreToolUse", "Notification"}
			},
			sources: []string{"claudecode.changelog"},
		},
		{
			name: "truncated",
			mutate: func(d *upstream.Digest) {
				for i := range 20 {
					d.Tools.Builtin = append(d.Tools.Builtin, "Tool"+strconv.Itoa(i))
				}
			},
			drift: true,
		},
	}
}

func TestDiffGoldens(t *testing.T) {
	t.Parallel()

	for _, tc := range diffCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			before := baselineDigest()
			after := baselineDigest()
			tc.mutate(after)
			after.Normalize()

			report, err := upstream.Diff(before, after)
			if err != nil {
				t.Fatalf("Diff() error = %v", err)
			}
			report.SourcesChanged = tc.sources

			if report.HasDrift() != tc.drift {
				t.Errorf("HasDrift() = %v, want %v", report.HasDrift(), tc.drift)
			}

			raw, err := report.JSON()
			if err != nil {
				t.Fatalf("JSON() error = %v", err)
			}

			checkGolden(t, tc.name+".txt", report.Text())
			checkGolden(t, tc.name+".md", report.Markdown())
			checkGolden(t, tc.name+".json", string(raw))
		})
	}
}

// checkGolden compares rendered output against a committed file, or
// rewrites it under -update.
func checkGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join(diffGoldenDir, name)

	if *updateGolden {
		if err := os.MkdirAll(diffGoldenDir, 0o750); err != nil {
			t.Fatalf("create golden dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run: go test ./internal/upstream -update)", path, err)
	}

	if got != string(want) {
		t.Errorf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

// TestDiffInformationalIsNotDrift is the property the exit code rests
// on: the three informational sections can all change without the tool
// reporting drift.
func TestDiffInformationalIsNotDrift(t *testing.T) {
	t.Parallel()

	before := baselineDigest()
	after := baselineDigest()
	after.Meta.DocsMaxMarker = "v2.1.280"
	after.SchemaStore.Settings.DefaultModes = []string{"acceptEdits"}
	after.Disagreements = []upstream.Disagreement{
		{Topic: "hooks.events", Source: "schemastore.plugin", DocsOnly: []string{"SessionStart"}},
	}
	after.Normalize()

	report, err := upstream.Diff(before, after)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	if report.HasDrift() {
		t.Errorf("HasDrift() = true, want false; changes = %+v", report.Changes)
	}
	if len(report.Informational) == 0 {
		t.Error("Informational is empty, want the three section changes")
	}
}

// TestDiffRowsDisambiguateByGroup covers the plugin manifest tables,
// which list one field name under more than one group. Keying rows by
// name alone would drop a row and report a change that did not happen.
func TestDiffRowsDisambiguateByGroup(t *testing.T) {
	t.Parallel()

	before := baselineDigest()
	after := baselineDigest()
	after.Plugins.ManifestFields[1].Type = "array"
	after.Normalize()

	report, err := upstream.Diff(before, after)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	if len(report.Changes) != 1 {
		t.Fatalf("Diff() = %d changes, want 1: %+v", len(report.Changes), report.Changes)
	}

	got := report.Changes[0]
	if got.Path != "plugins.manifest_fields" {
		t.Errorf("path = %q, want plugins.manifest_fields", got.Path)
	}
	if want := "hooks (metadata).type"; got.Item != want {
		t.Errorf("item = %q, want %q", got.Item, want)
	}
}

// TestDiffTruncatesLongLists keeps an issue body readable when a whole
// table arrives at once.
func TestDiffTruncatesLongLists(t *testing.T) {
	t.Parallel()

	before := baselineDigest()
	after := baselineDigest()
	for i := range 20 {
		after.Tools.Builtin = append(after.Tools.Builtin, "Tool"+strconv.Itoa(i))
	}
	after.Normalize()

	report, err := upstream.Diff(before, after)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	md := report.Markdown()
	if !strings.Contains(md, "... (20)") {
		t.Errorf("Markdown() did not truncate:\n%s", md)
	}
}

// TestDiffOfIdenticalDigestsIsEmpty is the merge-day case: a clean
// checkout diffed against itself reports nothing at all.
func TestDiffOfIdenticalDigestsIsEmpty(t *testing.T) {
	t.Parallel()

	report, err := upstream.Diff(baselineDigest(), baselineDigest())
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	if !report.Empty() {
		t.Errorf("Empty() = false, want true: %+v", report)
	}
	if got := report.Text(); got != "no drift\n" {
		t.Errorf("Text() = %q, want %q", got, "no drift\n")
	}
}

// TestDiffOfFixtureDigestWithItself exercises the real shape, which has
// nested maps, row arrays, and every section populated.
func TestDiffOfFixtureDigestWithItself(t *testing.T) {
	t.Parallel()

	report, err := upstream.Diff(fixtureDigest(t), fixtureDigest(t))
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}

	if !report.Empty() {
		t.Errorf("Diff() of a digest with itself = %+v, want empty", report)
	}
}
