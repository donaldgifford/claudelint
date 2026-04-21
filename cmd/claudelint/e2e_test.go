package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildBinary compiles the claudelint CLI into a temp path and returns
// the absolute path to the binary. Every E2E test starts from the
// same baseline, which costs one `go build` per test run. That's
// acceptable for a single-digit number of E2E tests.
func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "claudelint")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, ".")
	cmd.Env = os.Environ()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build: %v\n%s", err, stderr.String())
	}
	return bin
}

// fixtureDir returns the absolute path to the versioned fixture repo
// under testdata. Every E2E case walks this same tree so the expected
// diagnostics stay stable.
func fixtureDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("testdata/fixture")
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}
	return abs
}

// runBinary invokes the compiled binary with args, captures stdout and
// stderr, and returns them along with the exit code. The environment
// is scrubbed of NO_COLOR and color-related settings so color output
// is deterministic (no-TTY buffers always get uncolored text anyway,
// but belt-and-suspenders is cheap).
func runBinary(t *testing.T, bin string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=")
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run %s: %v", bin, err)
		}
		code = exitErr.ExitCode()
	}
	return out.String(), errBuf.String(), code
}

func TestE2ETextFormatDirtyFixture(t *testing.T) {
	bin := buildBinary(t)
	stdout, _, exit := runBinary(t, bin, "run", fixtureDir(t))

	if exit != 1 {
		t.Errorf("exit = %d, want 1 (dirty fixture has errors)", exit)
	}
	want := []string{
		"schema/frontmatter-required",
		"skills/trigger-clarity",
		"2 diagnostics, 1 files checked",
	}
	for _, w := range want {
		if !strings.Contains(stdout, w) {
			t.Errorf("stdout missing %q:\n%s", w, stdout)
		}
	}
}

func TestE2EJSONFormatShape(t *testing.T) {
	bin := buildBinary(t)
	stdout, _, exit := runBinary(t, bin, "run", "--format=json", fixtureDir(t))

	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}

	var report struct {
		SchemaVersion   string `json:"schema_version"`
		FilesChecked    int    `json:"files_checked"`
		DiagnosticCount int    `json:"diagnostic_count"`
		SeverityCount   struct {
			Error   int `json:"error"`
			Warning int `json:"warning"`
		} `json:"severity_count"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("parse JSON: %v\nstdout:\n%s", err, stdout)
	}
	if report.SchemaVersion != "1" {
		t.Errorf("schema_version = %q, want 1", report.SchemaVersion)
	}
	if report.DiagnosticCount != 2 {
		t.Errorf("diagnostic_count = %d, want 2", report.DiagnosticCount)
	}
	if report.SeverityCount.Error != 1 || report.SeverityCount.Warning != 1 {
		t.Errorf("severity_count = %+v, want error=1 warning=1", report.SeverityCount)
	}
}

func TestE2EGitHubFormatAnnotations(t *testing.T) {
	bin := buildBinary(t)
	stdout, _, exit := runBinary(t, bin, "run", "--format=github", fixtureDir(t))

	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	want := []string{
		"::error ",
		"::warning ",
		"title=schema/frontmatter-required",
		"title=skills/trigger-clarity",
	}
	for _, w := range want {
		if !strings.Contains(stdout, w) {
			t.Errorf("stdout missing %q:\n%s", w, stdout)
		}
	}
}

func TestE2EMaxWarningsPromotesToError(t *testing.T) {
	bin := buildBinary(t)
	// Disable the required-frontmatter rule so only the warning remains.
	// We pass --config pointing to a stub file.
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, ".claudelint.hcl")
	cfg := `claudelint { version = "1" }
rule "schema/frontmatter-required" { enabled = false }
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	// Without --max-warnings: clean exit despite one warning.
	_, _, exit := runBinary(t, bin, "run", "--config", cfgPath, fixtureDir(t))
	if exit != 0 {
		t.Errorf("exit = %d with 1 warning and no --max-warnings, want 0", exit)
	}

	// With --max-warnings=0: the warning promotes to a failing exit.
	_, _, exit = runBinary(t, bin, "run", "--config", cfgPath, "--max-warnings=0", fixtureDir(t))
	if exit != 1 {
		t.Errorf("exit = %d with --max-warnings=0, want 1", exit)
	}
}

func TestE2EQuietDropsNonErrors(t *testing.T) {
	bin := buildBinary(t)
	stdout, _, exit := runBinary(t, bin, "run", "--quiet", fixtureDir(t))

	if exit != 1 {
		t.Errorf("exit = %d, want 1 (still has error)", exit)
	}
	if strings.Contains(stdout, "skills/trigger-clarity") {
		t.Errorf("quiet still emitted warning:\n%s", stdout)
	}
	if !strings.Contains(stdout, "schema/frontmatter-required") {
		t.Errorf("quiet dropped the error:\n%s", stdout)
	}
	// Trailing summary line always prints — scripts grep it.
	if !strings.Contains(stdout, "files checked") {
		t.Errorf("quiet dropped summary line:\n%s", stdout)
	}
}

func TestE2EUnknownFormatUsageError(t *testing.T) {
	bin := buildBinary(t)
	_, stderr, exit := runBinary(t, bin, "run", "--format=bogus", fixtureDir(t))

	if exit != 2 {
		t.Errorf("exit = %d, want 2 (usage error)", exit)
	}
	if !strings.Contains(stderr, "unknown --format") {
		t.Errorf("stderr = %q, want it to mention unknown --format", stderr)
	}
}
