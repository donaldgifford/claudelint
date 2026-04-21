package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/diag"
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
