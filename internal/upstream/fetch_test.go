package upstream_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

// fastFetcher keeps retry backoff out of the test wall clock.
func fastFetcher(t *testing.T) *upstream.Fetcher {
	t.Helper()

	return upstream.NewFetcher(upstream.FetchOptions{
		Attempts: 3,
		Timeout:  2 * time.Second,
		Backoff:  time.Millisecond,
	})
}

func srcFor(id, rawURL string) upstream.Source {
	return upstream.Source{ID: id, URL: rawURL, Ext: "md", Tier: upstream.TierAuthoritative}
}

// assertFetchError checks that err is a *upstream.FetchError naming
// the given source and status.
func assertFetchError(t *testing.T, err error, source string, status int) {
	t.Helper()

	var fe *upstream.FetchError
	if !errors.As(err, &fe) {
		t.Fatalf("error is %T, want *upstream.FetchError", err)
	}
	if fe.Source != source {
		t.Errorf("FetchError.Source = %q, want %q", fe.Source, source)
	}
	if fe.Status != status {
		t.Errorf("FetchError.Status = %d, want %d", fe.Status, status)
	}
	if !errors.Is(err, upstream.ErrBadStatus) {
		t.Error("error does not wrap ErrBadStatus")
	}
}

func TestFetchWritesPagesAndManifest(t *testing.T) {
	t.Parallel()

	const body = "# Hooks\n\nsome docs\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"abc123"`)
		w.Header().Set("Last-Modified", "Fri, 11 Sep 2026 13:04:33 GMT")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	dir := t.TempDir()
	src := srcFor("docs.hooks", srv.URL)

	manifest, err := fastFetcher(t).Fetch(t.Context(), []upstream.Source{src}, dir)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "docs.hooks.md"))
	if err != nil {
		t.Fatalf("read fetched page: %v", err)
	}
	if string(got) != body {
		t.Errorf("page = %q, want %q", got, body)
	}

	rec, ok := manifest["docs.hooks"]
	if !ok {
		t.Fatal("manifest has no entry for docs.hooks")
	}
	if rec.Status != http.StatusOK {
		t.Errorf("Status = %d, want 200", rec.Status)
	}
	if rec.Bytes != int64(len(body)) {
		t.Errorf("Bytes = %d, want %d", rec.Bytes, len(body))
	}
	wantSHA := sha256.Sum256([]byte(body))
	if rec.SHA256 != hex.EncodeToString(wantSHA[:]) {
		t.Errorf("SHA256 = %q, want %x", rec.SHA256, wantSHA)
	}
	if rec.ETag != `"abc123"` {
		t.Errorf("ETag = %q", rec.ETag)
	}
	if rec.LastModified == "" {
		t.Error("LastModified is empty")
	}

	var onDisk upstream.Manifest
	raw, err := os.ReadFile(filepath.Join(dir, upstream.ManifestFile))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if onDisk["docs.hooks"].SHA256 != rec.SHA256 {
		t.Error("manifest on disk disagrees with the returned manifest")
	}
}

func TestFetchRetriesServerErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		failStatus   int
		failTimes    int32
		wantAttempts int32
		wantErr      bool
	}{
		{name: "recovers after one 503", failStatus: http.StatusServiceUnavailable, failTimes: 1, wantAttempts: 2},
		{name: "recovers after two 429s", failStatus: http.StatusTooManyRequests, failTimes: 2, wantAttempts: 3},
		{name: "gives up after the budget", failStatus: http.StatusBadGateway, failTimes: 5, wantAttempts: 3, wantErr: true},
		{name: "does not retry a 404", failStatus: http.StatusNotFound, failTimes: 5, wantAttempts: 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if calls.Add(1) <= tt.failTimes {
					w.WriteHeader(tt.failStatus)

					return
				}
				_, _ = w.Write([]byte("ok"))
			}))
			defer srv.Close()

			src := srcFor("docs.hooks", srv.URL)

			_, err := fastFetcher(t).Fetch(t.Context(), []upstream.Source{src}, t.TempDir())
			if tt.wantErr && err == nil {
				t.Fatal("Fetch succeeded, want an error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Fetch: %v", err)
			}
			if got := calls.Load(); got != tt.wantAttempts {
				t.Errorf("server saw %d requests, want %d", got, tt.wantAttempts)
			}

			if !tt.wantErr {
				return
			}
			assertFetchError(t, err, "docs.hooks", tt.failStatus)
		})
	}
}

func TestFetchFollowsRedirects(t *testing.T) {
	t.Parallel()

	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("final body"))
	}))
	defer final.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusMovedPermanently)
	}))
	defer redirector.Close()

	dir := t.TempDir()
	src := srcFor("schemastore.settings", redirector.URL)

	manifest, err := fastFetcher(t).Fetch(t.Context(), []upstream.Source{src}, dir)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	rec := manifest["schemastore.settings"]
	if rec.URL != final.URL {
		t.Errorf("manifest URL = %q, want the post-redirect %q", rec.URL, final.URL)
	}

	got, err := os.ReadFile(filepath.Join(dir, "schemastore.settings.md"))
	if err != nil {
		t.Fatalf("read page: %v", err)
	}
	if string(got) != "final body" {
		t.Errorf("body = %q", got)
	}
}

func TestFetchHonoursPerAttemptTimeout(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		_, _ = w.Write([]byte("too late"))
	}))
	defer srv.Close()
	defer close(release)

	f := upstream.NewFetcher(upstream.FetchOptions{
		Attempts: 1,
		Timeout:  50 * time.Millisecond,
		Backoff:  time.Millisecond,
	})

	_, err := f.Fetch(t.Context(), []upstream.Source{srcFor("docs.hooks", srv.URL)}, t.TempDir())
	if err == nil {
		t.Fatal("Fetch succeeded despite the timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want a deadline-exceeded cause", err)
	}
}

func TestFetchSendsUserAgent(t *testing.T) {
	t.Parallel()

	seen := make(chan string, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Header.Get("User-Agent")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	f := upstream.NewFetcher(upstream.FetchOptions{})
	if _, err := f.Fetch(t.Context(), []upstream.Source{srcFor("docs.hooks", srv.URL)}, t.TempDir()); err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	ua := <-seen
	if ua == "" || ua == "Go-http-client/1.1" {
		t.Errorf("User-Agent = %q, want one naming the repository", ua)
	}
}

func TestFetchToleratesOptionalSourceFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/probe" {
			w.WriteHeader(http.StatusFound)
			_, _ = w.Write([]byte("moved"))

			return
		}
		_, _ = w.Write([]byte("real page"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	srcs := []upstream.Source{
		srcFor("docs.hooks", srv.URL+"/docs"),
		{
			ID:       "probe.plugin_schema",
			URL:      srv.URL + "/probe",
			Ext:      "json",
			Tier:     upstream.TierCrossCheck,
			Optional: true,
		},
	}

	manifest, err := fastFetcher(t).Fetch(t.Context(), srcs, dir)
	if err != nil {
		t.Fatalf("Fetch: an optional source must not fail the run: %v", err)
	}

	probe, ok := manifest["probe.plugin_schema"]
	if !ok {
		t.Fatal("manifest has no record for the failed probe")
	}
	if probe.Status == http.StatusOK {
		t.Errorf("probe Status = %d, want the failure status", probe.Status)
	}
	if _, err := os.Stat(filepath.Join(dir, "probe.plugin_schema.json")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a failed probe must not leave a file behind")
	}
}

func TestLoadPages(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docs.hooks.md"), []byte("page"), 0o600); err != nil {
		t.Fatalf("seed page: %v", err)
	}

	srcs := []upstream.Source{
		srcFor("docs.hooks", "https://example.test/hooks.md"),
		{ID: "probe.plugin_schema", URL: "https://example.test/p.json", Ext: "json", Optional: true},
	}

	pages, err := upstream.LoadPages(dir, srcs)
	if err != nil {
		t.Fatalf("LoadPages: %v", err)
	}
	if string(pages["docs.hooks"]) != "page" {
		t.Errorf("docs.hooks = %q", pages["docs.hooks"])
	}
	if _, ok := pages["probe.plugin_schema"]; ok {
		t.Error("a missing optional source must be skipped, not synthesised")
	}

	required := []upstream.Source{srcFor("docs.tools", "https://example.test/tools.md")}
	if _, err := upstream.LoadPages(dir, required); err == nil {
		t.Error("LoadPages succeeded with a missing required page")
	}
}
