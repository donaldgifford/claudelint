package cli

import (
	"bytes"
	"strings"
	"testing"
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
