package upstream_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

// The validator runs a real subprocess, so these tests write a fake
// `claude` into a temp directory and point --claude at it. That keeps
// the suite offline, keeps it off any developer's installed CLI, and
// lets a test drive the two failure shapes that matter: output that is
// not the documented JSON, and output missing "success".
//
// The fake is a shell script, so these tests do not run on Windows.
// Neither does the repo's justfile, so that is not a new constraint.

// fakeClaude writes a script that prints body for any `plugin validate`
// invocation and a version string for `--version`, then returns its
// path. exitCode is what it exits with after a validate, which is how a
// test proves the exit code is not the signal.
func fakeClaude(t *testing.T, body string, exitCode int) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "claude")

	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "--version" ]; then
  echo "2.1.259 (Claude Code)"
  exit 0
fi
cat <<'PAYLOAD'
%s
PAYLOAD
exit %d
`, body, exitCode)

	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write the fake claude: %v", err)
	}

	return path
}

// validatorJSON builds one runtime response.
func validatorJSON(t *testing.T, success bool, contents int, errs, warns []string) string {
	t.Helper()

	msgs := func(in []string) []map[string]string {
		out := make([]map[string]string, 0, len(in))
		for _, m := range in {
			out = append(out, map[string]string{"path": "name", "message": m})
		}

		return out
	}

	payload := map[string]any{
		"success": success,
		"strict":  true,
		"target":  "/tmp/fixture/.claude-plugin/plugin.json",
		"manifest": map[string]any{
			"file":     "/tmp/fixture/.claude-plugin/plugin.json",
			"type":     "plugin",
			"errors":   msgs(errs),
			"warnings": msgs(warns),
		},
		"contents": make([]any, contents),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal the fake payload: %v", err)
	}

	return string(raw)
}

// oneFixture is the smallest fixture list a test can run.
func oneFixture(expect string) []upstream.RuntimeFixture {
	return []upstream.RuntimeFixture{{
		Expect: expect,
		Kind:   "plugin",
		Path:   "internal/artifact/testdata/ok/pluginroot",
	}}
}

func runValidate(t *testing.T, bin string, fixtures []upstream.RuntimeFixture) (*upstream.RuntimeReport, error) {
	t.Helper()

	return upstream.ValidateFixtures(context.Background(), &upstream.ValidateOptions{
		Claude:   bin,
		Root:     filepath.Join("..", ".."),
		Fixtures: fixtures,
		Timeout:  20 * time.Second,
	})
}

// TestValidateAgreesWhenTheRuntimeAgrees covers both directions of
// agreement, including the one that matters most: a fixture expected to
// fail, where the validator exits non-zero. If the exit code were read
// as the signal, this case would be reported as a tool failure.
func TestValidateAgreesWhenTheRuntimeAgrees(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expect   string
		success  bool
		exitCode int
	}{
		{"pass agrees with a clean run", upstream.ExpectPass, true, 0},
		{"fail agrees with a non-zero exit", upstream.ExpectFail, false, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bin := fakeClaude(t, validatorJSON(t, tc.success, 0, nil, nil), tc.exitCode)

			report, err := runValidate(t, bin, oneFixture(tc.expect))
			if err != nil {
				t.Fatalf("ValidateFixtures() error = %v", err)
			}
			if report.HasDisagreement() {
				t.Errorf("report claims a disagreement: %s", report.Text())
			}
			if report.ClaudeVersion != "2.1.259 (Claude Code)" {
				t.Errorf("ClaudeVersion = %q, want the runtime's own output", report.ClaudeVersion)
			}
		})
	}
}

// TestValidateReportsADisagreementVerbatim is the whole output of this
// command: the person reading it in CI cannot re-run the binary, so the
// runtime's own words have to reach them.
func TestValidateReportsADisagreementVerbatim(t *testing.T) {
	t.Parallel()

	body := validatorJSON(t, false, 0,
		[]string{"Invalid input: expected string, received undefined"},
		[]string{"No author information provided"})
	bin := fakeClaude(t, body, 1)

	report, err := runValidate(t, bin, oneFixture(upstream.ExpectPass))
	if err != nil {
		t.Fatalf("ValidateFixtures() error = %v", err)
	}

	if !report.HasDisagreement() {
		t.Fatal("a fixture expected to pass that the runtime rejected was not reported")
	}

	for _, rendering := range []string{report.Text(), report.Markdown()} {
		if !strings.Contains(rendering, "Invalid input: expected string, received undefined") {
			t.Errorf("the runtime's error text did not reach the report:\n%s", rendering)
		}
		if !strings.Contains(rendering, "No author information provided") {
			t.Errorf("the runtime's warning text did not reach the report:\n%s", rendering)
		}
	}

	raw, err := report.JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	if !strings.Contains(string(raw), "Invalid input") {
		t.Error("the JSON rendering dropped the runtime's message")
	}
}

// TestValidateProbeFailsWhenContentsAppear re-tests the assumption that
// bounds claudelint's coverage claim. The day the validator starts
// walking skills/, the report should say so.
func TestValidateProbeFailsWhenContentsAppear(t *testing.T) {
	t.Parallel()

	fixtures := oneFixture(upstream.ExpectPass)
	fixtures[0].Probe = upstream.ProbeContentsEmpty

	t.Run("empty contents agree", func(t *testing.T) {
		t.Parallel()

		bin := fakeClaude(t, validatorJSON(t, true, 0, nil, nil), 0)

		report, err := runValidate(t, bin, fixtures)
		if err != nil {
			t.Fatalf("ValidateFixtures() error = %v", err)
		}
		if report.HasDisagreement() {
			t.Errorf("an empty contents list was reported as a disagreement:\n%s", report.Text())
		}
	})

	t.Run("walked contents are reported", func(t *testing.T) {
		t.Parallel()

		bin := fakeClaude(t, validatorJSON(t, true, 3, nil, nil), 0)

		report, err := runValidate(t, bin, fixtures)
		if err != nil {
			t.Fatalf("ValidateFixtures() error = %v", err)
		}
		if !report.HasDisagreement() {
			t.Fatal("the validator walked contents and the probe stayed quiet")
		}
		if !strings.Contains(report.Markdown(), "walked 3 contents entries") {
			t.Errorf("the report does not say what the probe saw:\n%s", report.Markdown())
		}
	})
}

// TestValidateRefusesUndecidableOutput is the exit-2 boundary. Neither
// of these is a disagreement; both mean the run could not decide, and
// treating them as exit 1 would turn a broken tool into a false report
// about the fixtures.
func TestValidateRefusesUndecidableOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "output is not JSON",
			body: "npm ERR! could not determine executable to run",
			want: "not the documented JSON",
		},
		{
			name: "JSON has no success field",
			body: `{"strict":true,"manifest":{"errors":[],"warnings":[]}}`,
			want: `no "success" field`,
		},
		{
			name: "no output at all",
			body: "",
			want: "no output",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bin := fakeClaude(t, tc.body, 0)

			_, err := runValidate(t, bin, oneFixture(upstream.ExpectPass))
			if err == nil {
				t.Fatal("ValidateFixtures() accepted output it could not decide on")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error does not say what went wrong: %v", err)
			}
			if code := upstream.ExitCode(err); code != upstream.ExitFailure {
				t.Errorf("ExitCode() = %d, want %d", code, upstream.ExitFailure)
			}
		})
	}
}

// TestValidateFailsLoudlyWithoutABinary is the other exit-2 case.
func TestValidateFailsLoudlyWithoutABinary(t *testing.T) {
	t.Parallel()

	_, err := runValidate(t, filepath.Join(t.TempDir(), "not-here"), oneFixture(upstream.ExpectPass))
	if err == nil {
		t.Fatal("ValidateFixtures() ran without a validator")
	}
	if !strings.Contains(err.Error(), "--claude") {
		t.Errorf("error does not say how to fix it: %v", err)
	}
}

// TestValidateRejectsAnEscapingFixturePath keeps a fixture list from
// pointing the validator outside the repository.
func TestValidateRejectsAnEscapingFixturePath(t *testing.T) {
	t.Parallel()

	bin := fakeClaude(t, validatorJSON(t, true, 0, nil, nil), 0)
	fixtures := []upstream.RuntimeFixture{{
		Expect: upstream.ExpectPass,
		Kind:   "plugin",
		Path:   "../../../etc",
	}}

	if _, err := runValidate(t, bin, fixtures); err == nil {
		t.Fatal("ValidateFixtures() accepted a path outside the repository")
	}
}

// TestValidateHonoursTheTimeout proves a wedged validator costs one
// fixture rather than the run, and is reported as could-not-decide.
func TestValidateHonoursTheTimeout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo v; exit 0; fi\nsleep 30\n"
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatalf("write the fake claude: %v", err)
	}

	_, err := upstream.ValidateFixtures(context.Background(), &upstream.ValidateOptions{
		Claude:   bin,
		Root:     filepath.Join("..", ".."),
		Fixtures: oneFixture(upstream.ExpectPass),
		Timeout:  500 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("ValidateFixtures() waited out a wedged validator")
	}
	if code := upstream.ExitCode(err); code != upstream.ExitFailure {
		t.Errorf("ExitCode() = %d, want %d", code, upstream.ExitFailure)
	}
}

// TestExitCodeCoversEverySentinel is the trap the exit-code helper
// invites: a new sentinel silently becomes exit 2 unless it is added.
func TestExitCodeCoversEverySentinel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{"clean", nil, upstream.ExitClean},
		{"drift", upstream.ErrDrift, upstream.ExitDrift},
		{"runtime disagreement", upstream.ErrDisagreement, upstream.ExitDrift},
		{"stale page", upstream.ErrStale, upstream.ExitDrift},
		{"wrapped drift", fmt.Errorf("context: %w", upstream.ErrDrift), upstream.ExitDrift},
		{"anything else", errors.New("boom"), upstream.ExitFailure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := upstream.ExitCode(tc.err); got != tc.want {
				t.Errorf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

// TestRuntimeFixturesAreCanonical keeps the committed list formatted
// the way the digest and the lock are, and pointing at directories that
// exist.
func TestRuntimeFixturesAreCanonical(t *testing.T) {
	t.Parallel()

	raw := readFile(t, "runtime_fixtures.json")

	fixtures, err := upstream.DecodeRuntimeFixtures(raw)
	if err != nil {
		t.Fatalf("DecodeRuntimeFixtures() error = %v", err)
	}

	got, err := upstream.EncodeRuntimeFixtures(fixtures)
	if err != nil {
		t.Fatalf("EncodeRuntimeFixtures() error = %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Errorf("%s is not canonically formatted; rewrite it as:\n%s", upstream.RuntimeFixturesPath, got)
	}

	if len(fixtures) == 0 {
		t.Fatal("the fixture list is empty")
	}

	var probes int
	for _, f := range fixtures {
		if f.Expect != upstream.ExpectPass && f.Expect != upstream.ExpectFail {
			t.Errorf("%s: expect = %q, want pass or fail", f.Path, f.Expect)
		}
		if _, err := os.Stat(filepath.Join("..", "..", f.Path)); err != nil {
			t.Errorf("%s: %v", f.Path, err)
		}
		if f.Probe == upstream.ProbeContentsEmpty {
			probes++
		}
	}

	if probes == 0 {
		t.Error("no fixture carries the contents-empty probe, so the coverage assumption is untested")
	}
}
