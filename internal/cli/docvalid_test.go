package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// The catch-up in IMPL-0005 Phase 2 grew the canonical hook-event and
// tool lists. Growing a canonical list is the one change that turns a
// lint into a false positive for everybody at once, so the fixtures
// below declare every documented name and the run has to come back
// empty.
//
// The tree is copied into a temp directory before linting: the repo's
// own .claudelint.hcl ignores testdata/**, which would make a
// zero-diagnostic assertion vacuous if the fixtures were linted where
// they sit.

// docValidRoot copies the doc-valid fixture tree somewhere the repo's
// ignore globs do not reach and returns the copy's path.
func docValidRoot(t *testing.T) string {
	t.Helper()

	dst := filepath.Join(t.TempDir(), "repo")
	if err := os.CopyFS(dst, os.DirFS(filepath.Join("testdata", "docvalid"))); err != nil {
		t.Fatalf("copy fixtures: %v", err)
	}

	return dst
}

func TestDocValidFixturesLintClean(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"run", "--format=json", docValidRoot(t)})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() = %v, want nil", err)
	}

	var report struct {
		Diagnostics []struct {
			RuleID  string `json:"rule_id"`
			Path    string `json:"path"`
			Message string `json:"message"`
		} `json:"diagnostics"`
		FilesChecked int `json:"files_checked"`
	}
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, buf.String())
	}

	for _, d := range report.Diagnostics {
		t.Errorf("false positive: %s %s: %s", d.RuleID, d.Path, d.Message)
	}
	if report.FilesChecked == 0 {
		t.Fatal("no files were checked; the fixture tree did not survive the copy")
	}
}

// TestMarketplaceSourceFixtureIsAllNotices covers the archive and
// command source kinds end to end. They are doc-valid, so nothing may
// be reported as an error or a warning — but every one of them is
// unlintable content, so each entry should produce exactly one info
// notice saying so.
func TestMarketplaceSourceFixtureIsAllNotices(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "repo")
	if err := os.CopyFS(dst, os.DirFS(filepath.Join("testdata", "marketplace-sources"))); err != nil {
		t.Fatalf("copy fixtures: %v", err)
	}

	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"run", "--format=json", dst})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() = %v, want nil", err)
	}

	var report struct {
		Diagnostics []struct {
			RuleID   string `json:"rule_id"`
			Severity string `json:"severity"`
			Message  string `json:"message"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, buf.String())
	}

	if len(report.Diagnostics) != 4 {
		t.Fatalf("got %d diagnostics, want one per plugin entry: %+v",
			len(report.Diagnostics), report.Diagnostics)
	}
	for _, d := range report.Diagnostics {
		if d.RuleID != "marketplace/external-source-skipped" || d.Severity != "info" {
			t.Errorf("unexpected diagnostic: %s (%s) %s", d.RuleID, d.Severity, d.Message)
		}
	}
}

// TestDocValidFixturesCoverEveryHookEvent is what keeps the fixture
// honest. A clean run proves nothing if the fixture stopped mentioning
// the names that matter.
func TestDocValidFixturesCoverEveryHookEvent(t *testing.T) {
	raw := readDocValidFile(t, ".claude/hooks/every-event.json")

	var file struct {
		Hooks map[string]any `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	for event := range artifact.KnownHookEvents {
		if _, ok := file.Hooks[event]; !ok {
			t.Errorf("hook fixture does not declare %q", event)
		}
	}
	if len(file.Hooks) != len(artifact.KnownHookEvents) {
		t.Errorf("fixture declares %d events, the canonical list has %d",
			len(file.Hooks), len(artifact.KnownHookEvents))
	}
}

// TestDocValidFixturesCoverEveryTool pins the same property for tools,
// across the agent and command fixtures together. Superseded names are
// excluded on purpose: they belong in the deprecated-tool tests, and a
// fixture declaring one would stop being doc-valid.
func TestDocValidFixturesCoverEveryTool(t *testing.T) {
	files := map[string]string{
		"agent":   string(readDocValidFile(t, ".claude/agents/every-tool.md")),
		"command": string(readDocValidFile(t, ".claude/commands/every-tool.md")),
	}

	for tool := range artifact.KnownTools {
		if _, superseded := artifact.SupersededToolAdvice(tool); superseded {
			continue
		}

		for kind, body := range files {
			if !strings.Contains(body, "- "+tool+"\n") {
				t.Errorf("%s fixture does not declare %q", kind, tool)
			}
		}
	}
}

func readDocValidFile(t *testing.T, rel string) []byte {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", "docvalid", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	return raw
}
