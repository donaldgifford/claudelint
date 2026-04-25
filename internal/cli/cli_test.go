package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/config"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/discovery"
	"github.com/donaldgifford/claudelint/internal/engine"
	"github.com/donaldgifford/claudelint/internal/reporter"
)

func TestRootCmdSubcommands(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0.0.0-test", Commit: "deadbeef"})

	want := map[string]bool{
		"run":     false,
		"rules":   false,
		"init":    false,
		"version": false,
	}
	for _, sub := range root.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("root command missing subcommand %q", name)
		}
	}
}

func TestVersionCmdOutput(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v1.2.3", Commit: "abc1234"})

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() = %v, want nil", err)
	}

	got := stdout.String()
	// RulesetVersion and fingerprint change over time; assert the
	// shape but not the exact values.
	if !strings.HasPrefix(got, "claudelint v1.2.3 (abc1234)\nruleset    v") {
		t.Errorf("version output = %q, want prefix claudelint v1.2.3 ...", got)
	}
	if !strings.HasSuffix(got, ")\n") {
		t.Errorf("version output = %q, want trailing fingerprint paren", got)
	}
}

func TestBareInvocationAliasesToRun(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	// Point at an empty temp dir so the run has nothing to walk but
	// still reaches the reporter's summary line.
	root.SetArgs([]string{"run", t.TempDir()})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() = %v, want nil", err)
	}
	if !strings.Contains(stdout.String(), "diagnostics") {
		t.Errorf("bare invocation output = %q, want it to reach run stub", stdout.String())
	}
}

func TestRunUnknownFormatIsUsageError(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"run", "--format=foo", t.TempDir()})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute() = nil, want usage error for --format=foo")
	}
	if !strings.Contains(err.Error(), "unknown --format") {
		t.Errorf("error = %v, want it to mention unknown --format", err)
	}
}

func TestRunJSONFormatSmoke(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"run", "--format=json", t.TempDir()})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() = %v, want nil", err)
	}
	// JSON output is an object with schema_version. A substring check
	// is enough — the reporter package has the golden test.
	if !strings.Contains(buf.String(), `"schema_version"`) {
		t.Errorf("json output missing schema_version: %s", buf.String())
	}
}

func TestRunGitHubFormatSmoke(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"run", "--format=github", t.TempDir()})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() = %v, want nil", err)
	}
	// Empty-repo run prints only the trailing summary.
	if !strings.Contains(buf.String(), "0 diagnostics") {
		t.Errorf("github output missing summary: %s", buf.String())
	}
}

func TestExecuteExitCodeCleanRun(t *testing.T) {
	// Integration smoke over Execute: clean run returns exitSuccess.
	// We can't easily inject args into Execute (it uses os.Args via
	// cobra) so this test just verifies the Execute signature and
	// type — the behavioral matrix is covered by the helper tests
	// below.
	if exitSuccess != 0 {
		t.Errorf("exitSuccess = %d, want 0", exitSuccess)
	}
	if exitHasErrors != 1 {
		t.Errorf("exitHasErrors = %d, want 1", exitHasErrors)
	}
	if exitUsage != 2 {
		t.Errorf("exitUsage = %d, want 2", exitUsage)
	}
}

func TestRunFailedMatrix(t *testing.T) {
	// runFailed is the core exit-code decider; cover it exhaustively.
	cases := []struct {
		name     string
		errs     int
		warns    int
		maxWarns int
		want     bool
	}{
		{"clean", 0, 0, -1, false},
		{"one-error", 1, 0, -1, true},
		{"warnings-under-unlimited", 0, 5, -1, false},
		{"warnings-under-limit", 0, 3, 5, false},
		{"warnings-equal-limit", 0, 5, 5, false},
		{"warnings-over-limit", 0, 6, 5, true},
		{"warnings-over-zero-limit", 0, 1, 0, true},
		{"error-beats-limit", 1, 0, 5, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ds []diag.Diagnostic
			for i := 0; i < tc.errs; i++ {
				ds = append(ds, diag.Diagnostic{Severity: diag.SeverityError})
			}
			for i := 0; i < tc.warns; i++ {
				ds = append(ds, diag.Diagnostic{Severity: diag.SeverityWarning})
			}
			if got := runFailed(ds, tc.maxWarns); got != tc.want {
				t.Errorf("runFailed(errs=%d,warns=%d,max=%d) = %v, want %v",
					tc.errs, tc.warns, tc.maxWarns, got, tc.want)
			}
		})
	}
}

func TestRulesCmdListsEveryRule(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"rules"})

	if err := root.Execute(); err != nil {
		t.Fatalf("rules cmd: %v", err)
	}
	// The tabwriter header is stable.
	if !strings.Contains(buf.String(), "ID") || !strings.Contains(buf.String(), "CATEGORY") {
		t.Errorf("rules output missing header: %s", buf.String())
	}
	// At least one MVP rule ID should appear.
	if !strings.Contains(buf.String(), "schema/parse") {
		t.Errorf("rules output missing schema/parse: %s", buf.String())
	}
}

func TestRulesCmdDescribesSingleRule(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"rules", "schema/frontmatter-required"})

	if err := root.Execute(); err != nil {
		t.Fatalf("rules describe: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"schema/frontmatter-required", "schema", "error"} {
		if !strings.Contains(got, want) {
			t.Errorf("describe output missing %q: %s", want, got)
		}
	}
}

func TestRulesCmdUnknownRuleErrors(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"rules", "does/not-exist"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("describe of unknown rule should error")
	}
	if !strings.Contains(err.Error(), "unknown rule") {
		t.Errorf("error = %v, want unknown rule", err)
	}
}

func TestInitCmdWritesScaffold(t *testing.T) {
	dir := t.TempDir()
	// Change cwd to the tempdir so the init subcommand writes there.
	// Cobra's init implementation reads os.Getwd; isolating it here
	// avoids polluting the repo.
	revert := chdir(t, dir)
	defer revert()

	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"init"})

	if err := root.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}
	if !strings.Contains(buf.String(), ".claudelint.hcl") {
		t.Errorf("init output should mention filename: %s", buf.String())
	}
	if _, err := os.Stat(".claudelint.hcl"); err != nil {
		t.Errorf("init did not create .claudelint.hcl: %v", err)
	}

	// Second run without --force should be a refusal.
	buf.Reset()
	root.SetArgs([]string{"init"})
	if err := root.Execute(); err == nil {
		t.Errorf("second init without --force should error")
	}
}

func chdir(t *testing.T, dir string) func() {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("chdir back: %v", err)
		}
	}
}

func TestWriteSuppressedVerbose(t *testing.T) {
	var buf bytes.Buffer
	in := []engine.SuppressedDiagnostic{
		{
			Diagnostic: diag.Diagnostic{
				RuleID:  "skills/x",
				Path:    "a.md",
				Range:   diag.Range{Start: diag.Position{Line: 2, Column: 1}},
				Message: "boom",
			},
			Reason: "in-source ignore marker on line 2",
		},
		{
			Diagnostic: diag.Diagnostic{
				RuleID:  "schema/parse",
				Path:    ".claude/bad.json",
				Message: "invalid json",
			},
			Reason: "in-source ignore-file marker",
		},
	}
	if err := writeSuppressedVerbose(&buf, in); err != nil {
		t.Fatalf("writeSuppressedVerbose: %v", err)
	}
	got := buf.String()
	for _, w := range []string{"skills/x", "schema/parse", "2 diagnostic(s) suppressed"} {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q in:\n%s", w, got)
		}
	}
	// Empty list writes nothing.
	buf.Reset()
	if err := writeSuppressedVerbose(&buf, nil); err != nil {
		t.Fatalf("empty: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("empty list should print nothing, got: %q", buf.String())
	}
}

func TestAbsSkillDir(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"skills/foo/SKILL.md", "skills/foo"},
		{"/a/b/c/SKILL.md", "/a/b/c"},
		{"SKILL.md", "."},
		{`c:\a\b\SKILL.md`, `c:\a\b`},
	}
	for _, tc := range cases {
		if got := absSkillDir(tc.in); got != tc.want {
			t.Errorf("absSkillDir(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseOneDispatch(t *testing.T) {
	cases := []struct {
		name    string
		kind    artifact.ArtifactKind
		src     string
		wantErr bool
	}{
		{
			name: "claude_md",
			kind: artifact.KindClaudeMD,
			src:  "# Hello\n",
		},
		{
			name: "skill_missing_frontmatter",
			kind: artifact.KindSkill,
			src:  "body only\n",
			// ParseSkill may still succeed with empty frontmatter — we
			// just want to exercise the dispatch path.
		},
		{
			name:    "unknown_kind",
			kind:    artifact.ArtifactKind("bogus"),
			src:     "whatever",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := discovery.Candidate{
				Path: "x", AbsPath: "x", Kind: tc.kind,
			}
			_, perr := parseOne(c, []byte(tc.src))
			if tc.wantErr && perr == nil {
				t.Errorf("expected ParseError, got nil")
			}
			if !tc.wantErr && perr != nil {
				// Some parsers return ParseError for bad input — we
				// accept either outcome for dispatch coverage.
				t.Logf("parse returned %v (acceptable)", perr)
			}
		})
	}
}

func TestFilterIgnoredPaths(t *testing.T) {
	rc := config.Resolve(&config.File{
		Claudelint: &config.Claudelint{Version: "1"},
		Ignore:     &config.IgnoreBlock{Paths: []string{"testdata/**"}},
	})
	cands := []discovery.Candidate{
		{Path: "src/x.md"},
		{Path: "testdata/y.md"},
		{Path: "testdata/sub/z.md"},
	}
	out := filterIgnoredPaths(cands, rc)
	if len(out) != 1 || out[0].Path != "src/x.md" {
		t.Errorf("filterIgnoredPaths = %+v, want only src/x.md", out)
	}
}

func TestResolveConfigNilLoadResult(t *testing.T) {
	rc := resolveConfig(nil)
	if rc == nil {
		t.Fatalf("resolveConfig(nil) should not return nil")
	}
	if rc.Path() != "" {
		t.Errorf("path = %q, want empty", rc.Path())
	}
}

func TestWriteReportUnknownFormatPanics(t *testing.T) {
	// writeReport's switch has an "unreachable" default branch; hit it
	// by passing a format value validateFormat would have rejected.
	var buf bytes.Buffer
	err := writeReport(&buf, reporter.Summary{}, &runOptions{format: "nope"})
	if err == nil {
		t.Errorf("writeReport with bogus format should error")
	}
}

func TestFilterByQuiet(t *testing.T) {
	in := []diag.Diagnostic{
		{Severity: diag.SeverityError, RuleID: "a/e"},
		{Severity: diag.SeverityWarning, RuleID: "a/w"},
		{Severity: diag.SeverityInfo, RuleID: "a/i"},
	}
	// --quiet off: pass through.
	if got := filterByQuiet(in, false); len(got) != 3 {
		t.Errorf("quiet=false kept %d, want 3", len(got))
	}
	// --quiet on: only error remains.
	got := filterByQuiet(in, true)
	if len(got) != 1 || got[0].RuleID != "a/e" {
		t.Errorf("quiet=true = %+v, want only a/e", got)
	}
}
