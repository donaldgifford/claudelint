package upstream_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

// rewriteTransport sends every request to a test server, keeping the
// path. The source table holds absolute URLs across four hosts, so
// rewriting the host is what lets the whole pipeline run offline.
type rewriteTransport struct {
	base *url.URL
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = t.base.Scheme
	clone.URL.Host = t.base.Host
	clone.Host = ""

	return http.DefaultTransport.RoundTrip(clone)
}

// fixtureServer serves the assembled fixture pages at the paths of the
// real sources.
func fixtureServer(t *testing.T, pages map[string][]byte) *http.Client {
	t.Helper()

	byPath := make(map[string][]byte, len(pages))

	for _, s := range upstream.Sources() {
		u, err := url.Parse(s.URL)
		if err != nil {
			t.Fatalf("parse source URL %s: %v", s.URL, err)
		}
		if _, clash := byPath[u.Path]; clash {
			t.Fatalf("two sources share the path %s", u.Path)
		}
		if page, ok := pages[s.ID]; ok {
			byPath[u.Path] = page
		} else if !s.Optional {
			// A source no extractor reads still has to answer, or the
			// fetch fails before extraction begins.
			byPath[u.Path] = []byte("# placeholder\n")
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := byPath[r.URL.Path]
		if !ok {
			http.NotFound(w, r)

			return
		}
		w.Header().Set("Last-Modified", "Fri, 11 Sep 2026 13:04:33 GMT")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	return clientFor(t, srv.URL)
}

func clientFor(t *testing.T, base string) *http.Client {
	t.Helper()

	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	return &http.Client{Transport: rewriteTransport{base: u}}
}

// checkEnv is a temporary repository: a digest path, a lock path, and a
// client pointed at fixture content.
type checkEnv struct {
	Digest string
	Lock   string
	Opts   *upstream.CheckOptions
}

func newCheckEnv(t *testing.T) *checkEnv {
	t.Helper()

	dir := t.TempDir()
	env := &checkEnv{
		Digest: filepath.Join(dir, "digest.json"),
		Lock:   filepath.Join(dir, "sources.lock.json"),
	}
	env.Opts = &upstream.CheckOptions{
		Digest: env.Digest,
		Lock:   env.Lock,
		Format: "text",
		Fetch:  upstream.FetchOptions{Client: fixtureServer(t, fixturePages(t))},
	}

	return env
}

// bootstrap writes the first digest and lock, which is what
// "just spec-sync" does on a fresh branch.
func (e *checkEnv) bootstrap(t *testing.T) {
	t.Helper()

	e.Opts.Update = true
	defer func() { e.Opts.Update = false }()

	if err := upstream.Check(t.Context(), io.Discard, e.Opts); err != nil {
		t.Fatalf("Check(--update) error = %v", err)
	}
}

func (e *checkEnv) run(t *testing.T) (string, error) {
	t.Helper()

	var out bytes.Buffer
	err := upstream.Check(t.Context(), &out, e.Opts)

	return out.String(), err
}

// TestCheckExitsCleanAgainstACommittedDigest is the merge-day case: a
// digest written from the same sources it is compared against reports
// no drift and exits 0.
func TestCheckExitsCleanAgainstACommittedDigest(t *testing.T) {
	t.Parallel()

	env := newCheckEnv(t)
	env.bootstrap(t)

	out, err := env.run(t)
	if code := upstream.ExitCode(err); code != upstream.ExitClean {
		t.Fatalf("ExitCode() = %d, want %d (err = %v)", code, upstream.ExitClean, err)
	}
	if !strings.Contains(out, "no drift") {
		t.Errorf("Check() output = %q, want it to say no drift", out)
	}
}

// TestCheckUpdateIsDeterministic is the property the committed files
// rest on: a second --update run over unchanged sources rewrites
// identical bytes, so a no-op sync produces no diff.
func TestCheckUpdateIsDeterministic(t *testing.T) {
	t.Parallel()

	env := newCheckEnv(t)
	env.bootstrap(t)

	first := readFile(t, env.Digest)
	firstLock := readFile(t, env.Lock)

	env.bootstrap(t)

	if !bytes.Equal(first, readFile(t, env.Digest)) {
		t.Error("a second --update rewrote the digest differently")
	}
	if !bytes.Equal(firstLock, readFile(t, env.Lock)) {
		t.Error("a second --update rewrote the lock differently")
	}
}

// TestCheckExitsOneOnDrift removes a hook event from the committed
// digest, which is exactly the scratch-commit check the workflow is
// validated with before merge.
func TestCheckExitsOneOnDrift(t *testing.T) {
	t.Parallel()

	env := newCheckEnv(t)
	env.bootstrap(t)

	removeHookEvent(t, env.Digest, "PreModelSwitch")

	env.Opts.Format = "markdown"

	out, err := env.run(t)
	if code := upstream.ExitCode(err); code != upstream.ExitDrift {
		t.Fatalf("ExitCode() = %d, want %d (err = %v)", code, upstream.ExitDrift, err)
	}
	for _, want := range []string{"hooks.events", "PreModelSwitch", "added"} {
		if !strings.Contains(out, want) {
			t.Errorf("report does not mention %q:\n%s", want, out)
		}
	}
}

// TestCheckExitsTwoWhenASourceIsUnavailable covers the failure code: a
// source that will not answer leaves the tool unable to decide, which
// is not the same as no drift.
func TestCheckExitsTwoWhenASourceIsUnavailable(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	opts := &upstream.CheckOptions{
		Digest: filepath.Join(dir, "digest.json"),
		Lock:   filepath.Join(dir, "sources.lock.json"),
		Format: "text",
		Fetch:  upstream.FetchOptions{Client: clientFor(t, srv.URL), Attempts: 1},
	}

	err := upstream.Check(t.Context(), io.Discard, opts)
	if code := upstream.ExitCode(err); code != upstream.ExitFailure {
		t.Fatalf("ExitCode() = %d, want %d (err = %v)", code, upstream.ExitFailure, err)
	}
}

// TestCheckKeepsTheWorkDirectory covers --work, which the workflow uses
// to upload the fetched pages when a run fails.
func TestCheckKeepsTheWorkDirectory(t *testing.T) {
	t.Parallel()

	env := newCheckEnv(t)
	work := filepath.Join(t.TempDir(), "work")
	env.Opts.Work = work
	env.bootstrap(t)

	entries, err := os.ReadDir(work)
	if err != nil {
		t.Fatalf("read work directory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("work directory is empty, want the fetched sources")
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if !slices.Contains(names, "manifest.json") {
		t.Errorf("work directory = %v, want a manifest.json", names)
	}
}

// TestCheckWritesTheReportToAFile covers --out, which feeds the step
// summary and the issue body.
func TestCheckWritesTheReportToAFile(t *testing.T) {
	t.Parallel()

	env := newCheckEnv(t)
	env.bootstrap(t)

	report := filepath.Join(t.TempDir(), "report.md")
	env.Opts.Format = "markdown"
	env.Opts.Out = report

	out, err := env.run(t)
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !strings.Contains(out, report) {
		t.Errorf("stdout = %q, want it to name the report file", out)
	}
	if body := string(readFile(t, report)); !strings.Contains(body, "No drift") {
		t.Errorf("report = %q, want the no-drift line", body)
	}
}

// TestDigestSubcommandRunsOffline checks the pull/digest split: a work
// directory left by an earlier run is enough to rebuild both committed
// files without touching the network.
func TestDigestSubcommandRunsOffline(t *testing.T) {
	t.Parallel()

	env := newCheckEnv(t)
	work := filepath.Join(t.TempDir(), "work")
	env.Opts.Work = work
	env.bootstrap(t)

	dir := t.TempDir()
	digest := filepath.Join(dir, "digest.json")
	lock := filepath.Join(dir, "sources.lock.json")

	code := run(t, "digest", "--in", work, "--out", digest, "--lock", lock)
	if code != upstream.ExitClean {
		t.Fatalf("specdrift digest exited %d, want %d", code, upstream.ExitClean)
	}

	if !bytes.Equal(readFile(t, digest), readFile(t, env.Digest)) {
		t.Error("digest subcommand produced different bytes from check --update")
	}
}

// TestDigestExitsTwoWhenAnAnchorIsGone is the third Phase 1 success
// criterion: a page that lost the heading an extractor scopes to makes
// the tool exit 2, name the source and the anchor, and write nothing.
// Exiting 0 with a digest missing that section would diff as upstream
// deleting a feature.
func TestDigestExitsTwoWhenAnAnchorIsGone(t *testing.T) {
	t.Parallel()

	env := newCheckEnv(t)
	work := filepath.Join(t.TempDir(), "work")
	env.Opts.Work = work
	env.bootstrap(t)

	gutPage(t, filepath.Join(work, "docs.skills.md"), "### Frontmatter reference")

	digest := filepath.Join(t.TempDir(), "digest.json")
	lock := filepath.Join(t.TempDir(), "sources.lock.json")

	var stderr bytes.Buffer
	code := upstream.Execute(
		[]string{"digest", "--in", work, "--out", digest, "--lock", lock},
		io.Discard, &stderr,
	)

	if code != upstream.ExitFailure {
		t.Fatalf("specdrift digest exited %d, want %d", code, upstream.ExitFailure)
	}
	for _, want := range []string{"docs.skills", "Frontmatter reference", "anchor"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr does not mention %q:\n%s", want, stderr.String())
		}
	}
	if _, err := os.Stat(digest); !os.IsNotExist(err) {
		t.Errorf("digest was written despite the failure (stat err = %v)", err)
	}
}

// gutPage removes a heading line from a fetched page, simulating an
// upstream edit that renames or drops a section.
func gutPage(t *testing.T, path, heading string) {
	t.Helper()

	lines := strings.Split(string(readFile(t, path)), "\n")
	kept := make([]string, 0, len(lines))
	found := false

	for _, ln := range lines {
		if strings.TrimSpace(ln) == heading {
			found = true

			continue
		}
		kept = append(kept, ln)
	}

	if !found {
		t.Fatalf("page %s has no heading %q", path, heading)
	}
	if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestDiffSubcommandExitCodes pins the mapping the workflow depends on,
// without any network at all.
func TestDiffSubcommandExitCodes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	base := filepath.Join(dir, "base.json")
	head := filepath.Join(dir, "head.json")
	same := filepath.Join(dir, "same.json")

	writeDigest(t, base, baselineDigest())
	writeDigest(t, same, baselineDigest())

	drifted := baselineDigest()
	drifted.Hooks.Events = append(drifted.Hooks.Events, "PreModelSwitch")
	writeDigest(t, head, drifted)

	tests := []struct {
		name string
		args []string
		want int
	}{
		{"identical", []string{"diff", "--base", base, "--head", same}, upstream.ExitClean},
		{"drift", []string{"diff", "--base", base, "--head", head}, upstream.ExitDrift},
		{
			"missing file",
			[]string{"diff", "--base", base, "--head", filepath.Join(dir, "nope.json")},
			upstream.ExitFailure,
		},
		{
			"unknown format",
			[]string{"diff", "--base", base, "--head", same, "--format", "yaml"},
			upstream.ExitFailure,
		},
		{"missing required flag", []string{"diff", "--base", base}, upstream.ExitFailure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := run(t, tc.args...); got != tc.want {
				t.Errorf("specdrift %v exited %d, want %d", tc.args, got, tc.want)
			}
		})
	}
}

// TestRootCommandHasEverySubcommand guards the wiring, which is easy to
// break and invisible until a workflow step fails.
func TestRootCommandHasEverySubcommand(t *testing.T) {
	t.Parallel()

	root := upstream.NewRootCommand()

	names := make([]string, 0, len(root.Commands()))
	for _, c := range root.Commands() {
		names = append(names, c.Name())
	}
	slices.Sort(names)

	want := []string{"check", "diff", "digest", "pull"}
	if !slices.Equal(names, want) {
		t.Errorf("subcommands = %v, want %v", names, want)
	}
}

// run executes the command tree and returns its exit code.
func run(t *testing.T, args ...string) int {
	t.Helper()

	return upstream.Execute(args, io.Discard, io.Discard)
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return raw
}

func writeDigest(t *testing.T, path string, d *upstream.Digest) {
	t.Helper()

	raw, err := d.Encode()
	if err != nil {
		t.Fatalf("encode digest: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// removeHookEvent drops one event from a committed digest, simulating
// upstream adding it.
func removeHookEvent(t *testing.T, path, event string) {
	t.Helper()

	d, err := upstream.Decode(readFile(t, path))
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}

	i := slices.Index(d.Hooks.Events, event)
	if i < 0 {
		t.Fatalf("digest has no hook event %q", event)
	}
	d.Hooks.Events = slices.Delete(d.Hooks.Events, i, i+1)

	writeDigest(t, path, d)
}
