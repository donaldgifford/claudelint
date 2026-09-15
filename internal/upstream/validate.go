package upstream

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The digest says what the documentation claims. This says what the
// runtime actually does, which is not the same question.
//
// claudelint's fixtures encode a belief about what Claude Code accepts.
// The Claude Code CLI can answer that directly: `claude plugin validate
// --strict --json` reports success, errors, and warnings for a plugin
// or marketplace directory. Running it over the committed fixtures
// turns "we think this is valid" into "the runtime agreed on the day
// this ran".
//
// The disagreements are the output. A fixture claudelint calls valid
// that the runtime rejects is a claudelint bug; one the runtime accepts
// that claudelint rejects is a rule that is too strict. Either way the
// report names the fixture and quotes the runtime verbatim, because the
// person reading it in CI cannot re-run the binary.

//go:embed runtime_fixtures.json
var embeddedRuntimeFixtures []byte

// RuntimeFixturesPath is the committed fixture list.
const RuntimeFixturesPath = "internal/upstream/runtime_fixtures.json"

// Expectations a fixture can carry.
const (
	// ExpectPass means the runtime should validate the fixture
	// cleanly under --strict.
	ExpectPass = "pass"
	// ExpectFail means the runtime should reject it.
	ExpectFail = "fail"
)

// ProbeContentsEmpty asserts that the runtime reported no walked
// contents. It exists because claudelint's coverage claim depends on
// the validator not descending into skills/: the day it does, the
// report should say so rather than the assumption quietly aging.
const ProbeContentsEmpty = "contents-empty"

// defaultValidateTimeout bounds one fixture, not the whole run. A
// validator that hangs on one directory should not cost the rest.
const defaultValidateTimeout = 60 * time.Second

// ErrDisagreement is the sentinel behind exit 1 for validate-fixtures:
// the runtime and claudelint disagree about at least one fixture.
var ErrDisagreement = errors.New("runtime validator disagrees with a committed fixture")

// RuntimeFixture is one directory to hand the runtime validator, and
// what the result should be. Fields are declared in json-tag order so
// the encoded file is byte-stable.
type RuntimeFixture struct {
	// Expect is ExpectPass or ExpectFail.
	Expect string `json:"expect"`
	// Kind is "plugin" or "marketplace", for the report's grouping.
	Kind string `json:"kind"`
	// Note says why this fixture is in the list.
	Note string `json:"note,omitempty"`
	// Path is repo-relative.
	Path string `json:"path"`
	// Probe names an extra assertion beyond success, or is empty.
	Probe string `json:"probe,omitempty"`
}

// runtimeMessage is one error or warning as the validator reports it.
type runtimeMessage struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

// String renders a message the way the report quotes it.
func (m runtimeMessage) String() string {
	if m.Path == "" {
		return m.Message
	}

	return m.Path + ": " + m.Message
}

// runtimeManifest is the per-manifest half of the validator's JSON.
type runtimeManifest struct {
	Errors   []runtimeMessage `json:"errors"`
	File     string           `json:"file"`
	Type     string           `json:"type"`
	Warnings []runtimeMessage `json:"warnings"`
}

// runtimeOutput is the shape of `claude plugin validate --json`.
//
// Success is a pointer on purpose. An empty object decodes into a bool
// as false, which is indistinguishable from a genuine rejection — so
// the day the runtime changes its output shape, every passing fixture
// would be reported as a disagreement. A nil Success means "could not
// tell", which is exit 2, not exit 1.
type runtimeOutput struct {
	Contents []json.RawMessage `json:"contents"`
	Manifest runtimeManifest   `json:"manifest"`
	Strict   bool              `json:"strict"`
	Success  *bool             `json:"success"`
	Target   string            `json:"target"`
}

// RuntimeResult is one fixture's outcome.
type RuntimeResult struct {
	// Agrees is false when the runtime and the fixture disagree.
	Agrees bool `json:"agrees"`
	// Contents is how many entries the validator walked.
	Contents int `json:"contents"`
	// Errors and Warnings are the runtime's own text, verbatim.
	Errors []string `json:"errors,omitempty"`
	// Expect is the fixture's declared expectation.
	Expect string `json:"expect"`
	// Kind is the fixture's artifact kind.
	Kind string `json:"kind"`
	// Note carries the fixture's reason forward into the report.
	Note string `json:"note,omitempty"`
	// Path is the fixture path.
	Path string `json:"path"`
	// Probe is the extra assertion, if any, and ProbeOK its outcome.
	Probe    string   `json:"probe,omitempty"`
	ProbeOK  bool     `json:"probe_ok,omitempty"`
	Success  bool     `json:"success"`
	Warnings []string `json:"warnings,omitempty"`
}

// RuntimeReport is every fixture's outcome plus the runtime that
// produced them.
type RuntimeReport struct {
	// ClaudeVersion is `claude --version` verbatim, so a report read
	// months later says which runtime agreed.
	ClaudeVersion string `json:"claude_version"`
	// Results are sorted by path.
	Results []RuntimeResult `json:"results"`
}

// ValidateOptions configures one validate-fixtures run.
type ValidateOptions struct {
	// Claude is the validator binary. Empty means look it up on PATH.
	Claude string
	// Root is the directory fixture paths resolve against.
	Root string
	// Fixtures overrides the embedded list. Nil means use it.
	Fixtures []RuntimeFixture
	// Env is the child environment. Nil means a minimal allowlist.
	// Inheriting the caller's environment would let a developer's
	// ANTHROPIC_* or CLAUDE_CODE_* settings change the answer.
	Env []string
	// Timeout bounds one fixture. Zero means the default.
	Timeout time.Duration
}

// ValidateError names everything a failing run needs so nobody has to
// download a CI artifact to understand it.
type ValidateError struct {
	Binary string
	Path   string
	Stderr string
	Err    error
}

func (e *ValidateError) Error() string {
	msg := fmt.Sprintf("%s: validating %s: %v", e.Binary, e.Path, e.Err)
	if e.Stderr != "" {
		msg += "\nstderr: " + e.Stderr
	}

	return msg
}

func (e *ValidateError) Unwrap() error { return e.Err }

// EmbeddedRuntimeFixtures returns the committed fixture list.
func EmbeddedRuntimeFixtures() ([]RuntimeFixture, error) {
	return DecodeRuntimeFixtures(embeddedRuntimeFixtures)
}

// DecodeRuntimeFixtures parses a fixture list.
func DecodeRuntimeFixtures(data []byte) ([]RuntimeFixture, error) {
	var out []RuntimeFixture
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode %s: %w", RuntimeFixturesPath, err)
	}

	return out, nil
}

// EncodeRuntimeFixtures returns the canonical bytes, so the committed
// file is formatted the way the digest and the lock are.
func EncodeRuntimeFixtures(fixtures []RuntimeFixture) ([]byte, error) {
	var b strings.Builder
	if err := encodeJSON(&b, fixtures); err != nil {
		return nil, err
	}

	return []byte(b.String()), nil
}

// ValidateFixtures runs the Claude Code validator over every fixture
// and reports where it and claudelint disagree.
//
// An error means the run could not decide: the binary was missing, a
// fixture path escaped the root, or the validator produced something
// that is not its documented JSON. A disagreement is not an error — it
// is a result, and the caller turns it into exit 1.
func ValidateFixtures(ctx context.Context, opts *ValidateOptions) (*RuntimeReport, error) {
	bin, err := exec.LookPath(orDefault(opts.Claude, "claude"))
	if err != nil {
		return nil, fmt.Errorf("locate the claude binary (pass --claude): %w", err)
	}

	fixtures := opts.Fixtures
	if fixtures == nil {
		if fixtures, err = EmbeddedRuntimeFixtures(); err != nil {
			return nil, err
		}
	}

	configDir, cleanup, err := emptyConfigDir()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	env := opts.Env
	if env == nil {
		env = minimalEnv()
	}
	env = append(slices.Clone(env), "CLAUDE_CONFIG_DIR="+configDir)

	version, err := runtimeVersion(ctx, bin, env, opts.Timeout)
	if err != nil {
		return nil, err
	}

	report := &RuntimeReport{ClaudeVersion: version, Results: make([]RuntimeResult, 0, len(fixtures))}
	for i := range fixtures {
		result, rerr := validateOne(ctx, bin, env, opts, &fixtures[i])
		if rerr != nil {
			return nil, rerr
		}
		report.Results = append(report.Results, result)
	}

	slices.SortFunc(report.Results, func(a, b RuntimeResult) int {
		return strings.Compare(a.Path, b.Path)
	})

	return report, nil
}

// validateOne runs the validator against a single fixture.
func validateOne(
	ctx context.Context,
	bin string,
	env []string,
	opts *ValidateOptions,
	f *RuntimeFixture,
) (RuntimeResult, error) {
	target, err := fixturePath(opts.Root, f.Path)
	if err != nil {
		return RuntimeResult{}, err
	}

	stdout, stderr, err := runClaude(ctx, bin, env, opts.Timeout,
		"plugin", "validate", "--strict", "--json", target)

	// The validator exits non-zero whenever a fixture fails, which is
	// the expected outcome for every "fail" entry. So the exit code is
	// not the signal: decode stdout first, and only consult the error
	// when there is nothing to decode.
	var out runtimeOutput
	if jerr := json.Unmarshal(stdout, &out); jerr != nil || out.Success == nil {
		return RuntimeResult{}, &ValidateError{
			Binary: bin,
			Path:   f.Path,
			Stderr: excerpt(stderr),
			Err:    decodeFailure(jerr, err, stdout),
		}
	}

	return resultFor(f, &out), nil
}

// resultFor turns the validator's output into one report row.
func resultFor(f *RuntimeFixture, out *runtimeOutput) RuntimeResult {
	success := *out.Success

	result := RuntimeResult{
		Contents: len(out.Contents),
		Errors:   messageStrings(out.Manifest.Errors),
		Expect:   f.Expect,
		Kind:     f.Kind,
		Note:     f.Note,
		Path:     f.Path,
		Probe:    f.Probe,
		Success:  success,
		Warnings: messageStrings(out.Manifest.Warnings),
	}

	result.Agrees = success == (f.Expect == ExpectPass)
	if f.Probe == ProbeContentsEmpty {
		result.ProbeOK = len(out.Contents) == 0
		result.Agrees = result.Agrees && result.ProbeOK
	}

	return result
}

// runClaude runs one validator invocation with its own timeout.
//
// stdout and stderr are captured separately: node and npm write
// warnings to stderr, and interleaving them into stdout would destroy
// the JSON parse.
func runClaude(
	ctx context.Context,
	bin string,
	env []string,
	timeout time.Duration,
	args ...string,
) (stdout, stderr []byte, err error) {
	if timeout <= 0 {
		timeout = defaultValidateTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var outBuf, errBuf bytes.Buffer

	//nolint:gosec // G204: bin comes from --claude, a developer-supplied flag on a dev tool.
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	// A child that ignores the kill on cancel must not wedge Wait.
	cmd.WaitDelay = 5 * time.Second

	err = cmd.Run()

	return outBuf.Bytes(), errBuf.Bytes(), err
}

// runtimeVersion records which runtime agreed, so a report read months
// later still means something.
func runtimeVersion(ctx context.Context, bin string, env []string, timeout time.Duration) (string, error) {
	stdout, stderr, err := runClaude(ctx, bin, env, timeout, "--version")
	if err != nil {
		return "", &ValidateError{Binary: bin, Path: "--version", Stderr: excerpt(stderr), Err: err}
	}

	return strings.TrimSpace(string(stdout)), nil
}

// decodeFailure says which of the two things went wrong: the validator
// produced something unparseable, or it produced nothing at all.
func decodeFailure(jerr, runErr error, stdout []byte) error {
	if len(bytes.TrimSpace(stdout)) == 0 {
		if runErr != nil {
			return fmt.Errorf("no output: %w", runErr)
		}

		return errors.New("no output")
	}
	if jerr != nil {
		return fmt.Errorf("output is not the documented JSON (%w): %s", jerr, excerpt(stdout))
	}

	return fmt.Errorf(`output has no "success" field: %s`, excerpt(stdout))
}

// fixturePath resolves a fixture against the root and refuses one that
// escapes it.
func fixturePath(root, rel string) (string, error) {
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("fixture path %q must be relative and stay inside the repository", rel)
	}
	if root == "" {
		root = "."
	}

	abs, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", rel, err)
	}

	return abs, nil
}

// emptyConfigDir gives the validator a config directory of its own.
// CLAUDE_CONFIG_DIR set to the empty string reads as unset and falls
// back to the developer's real ~/.claude, which would make a local run
// and a CI run disagree for reasons that have nothing to do with the
// fixtures.
func emptyConfigDir() (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "specdrift-config-")
	if err != nil {
		return "", nil, fmt.Errorf("create an empty config directory: %w", err)
	}

	return dir, func() {
		// A leftover temp directory is not worth failing a run over.
		if rerr := os.RemoveAll(dir); rerr != nil {
			fmt.Fprintf(os.Stderr, "specdrift: could not remove %s: %v\n", dir, rerr)
		}
	}, nil
}

// minimalEnv is what the validator needs and nothing more. Inheriting
// the caller's environment would let ANTHROPIC_* or CLAUDE_CODE_*
// settings change the answer.
func minimalEnv() []string {
	var env []string
	for _, key := range []string{"PATH", "HOME", "TMPDIR", "LANG"} {
		if v, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+v)
		}
	}

	return env
}

// messageStrings renders the runtime's messages for the report.
func messageStrings(msgs []runtimeMessage) []string {
	if len(msgs) == 0 {
		return nil
	}

	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.String())
	}

	return out
}

// excerpt trims captured output to something a CI log can carry.
func excerpt(b []byte) string {
	const limit = 400

	s := strings.TrimSpace(string(b))
	if len(s) <= limit {
		return s
	}

	return s[:limit] + "…"
}

// orDefault returns value when it is set, and fallback otherwise.
func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}
