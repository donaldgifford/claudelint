package upstream

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ManifestFile is the name of the fetch manifest written into the work
// directory alongside the fetched pages.
const ManifestFile = "manifest.json"

// Fetch defaults. Timeout is per attempt rather than per source so a
// retry after a slow response still has a full budget.
const (
	defaultAttempts  = 3
	defaultTimeout   = 30 * time.Second
	defaultBackoff   = 500 * time.Millisecond
	defaultUserAgent = "claudelint-specdrift (+https://github.com/donaldgifford/claudelint)"
)

// FetchError reports that a source could not be retrieved. Every field
// is named in Error so a failing workflow run identifies the source
// without opening the artifact.
type FetchError struct {
	// Source is the source id.
	Source string
	// URL is the URL that was requested.
	URL string
	// Attempts is how many times the request was tried.
	Attempts int
	// Status is the last HTTP status observed, or 0 if no response
	// arrived.
	Status int
	// Reason is the underlying error.
	Reason error
}

func (e *FetchError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("fetch %s (%s): %d after %d attempts: %v",
			e.Source, e.URL, e.Status, e.Attempts, e.Reason)
	}

	return fmt.Sprintf("fetch %s (%s): after %d attempts: %v",
		e.Source, e.URL, e.Attempts, e.Reason)
}

// Unwrap exposes the underlying transport or status error.
func (e *FetchError) Unwrap() error { return e.Reason }

// ErrBadStatus is the reason recorded when a source answers with a
// status that is not 200.
var ErrBadStatus = errors.New("unexpected http status")

// FetchRecord is one entry of the fetch manifest. Keys are sorted on
// marshal because Manifest is a map, so two runs over unchanged
// sources produce byte-identical manifests.
type FetchRecord struct {
	// Bytes is the response body length.
	Bytes int64 `json:"bytes"`
	// ETag is the response ETag header, if any.
	ETag string `json:"etag,omitempty"`
	// LastModified is the response Last-Modified header, if any.
	LastModified string `json:"last_modified,omitempty"`
	// SHA256 is the hex digest of the response body.
	SHA256 string `json:"sha256"`
	// Status is the final HTTP status code, or 0 when no response
	// arrived at all.
	Status int `json:"status"`
	// URL is the final URL after redirects.
	URL string `json:"url"`
}

// Manifest maps a source id to what was fetched for it.
type Manifest map[string]FetchRecord

// FetchOptions tunes a Fetcher. The zero value is usable: missing
// fields fall back to three attempts, a thirty-second per-attempt
// timeout, a five-hundred-millisecond initial backoff, and a
// User-Agent naming this repository.
type FetchOptions struct {
	// Client is the HTTP client. A nil client uses a fresh
	// http.Client, which follows up to ten redirects and imposes no
	// client-level deadline; the per-attempt deadline comes from the
	// request context instead.
	Client *http.Client
	// UserAgent is sent with every request.
	UserAgent string
	// Attempts is the total number of tries per source, including the
	// first.
	Attempts int
	// Timeout bounds a single attempt.
	Timeout time.Duration
	// Backoff is the delay before the second attempt; it doubles
	// before each subsequent one.
	Backoff time.Duration
}

func (o FetchOptions) withDefaults() FetchOptions {
	if o.Client == nil {
		o.Client = &http.Client{}
	}
	if o.UserAgent == "" {
		o.UserAgent = defaultUserAgent
	}
	if o.Attempts <= 0 {
		o.Attempts = defaultAttempts
	}
	if o.Timeout <= 0 {
		o.Timeout = defaultTimeout
	}
	if o.Backoff <= 0 {
		o.Backoff = defaultBackoff
	}

	return o
}

// Fetcher retrieves upstream sources into a work directory.
type Fetcher struct {
	opts FetchOptions
}

// NewFetcher returns a Fetcher using opts, with unset fields filled in
// from the package defaults.
func NewFetcher(opts FetchOptions) *Fetcher {
	return &Fetcher{opts: opts.withDefaults()}
}

// Fetch retrieves every source into workDir as "<id>.<ext>" and writes
// the manifest. A required source that still fails after every attempt
// returns a *FetchError and no manifest; an optional source records
// what happened and the run continues.
func (f *Fetcher) Fetch(ctx context.Context, srcs []Source, workDir string) (Manifest, error) {
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return nil, fmt.Errorf("create work directory: %w", err)
	}

	manifest := make(Manifest, len(srcs))

	for _, s := range srcs {
		body, rec, err := f.fetchSource(ctx, s)
		if err != nil {
			if !s.Optional {
				return nil, err
			}
			manifest[s.ID] = rec

			continue
		}

		path := filepath.Join(workDir, s.File())
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", path, err)
		}
		manifest[s.ID] = rec
	}

	if err := writeJSONFile(filepath.Join(workDir, ManifestFile), manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

// fetchSource retries one source until it succeeds or the attempt
// budget runs out. On failure it still returns the record it managed
// to build so an optional source can be logged.
func (f *Fetcher) fetchSource(ctx context.Context, s Source) ([]byte, FetchRecord, error) {
	var (
		lastErr    error
		lastStatus int
		backoff    = f.opts.Backoff
	)

	for attempt := 1; attempt <= f.opts.Attempts; attempt++ {
		if attempt > 1 {
			if err := sleepCtx(ctx, backoff); err != nil {
				lastErr = err

				break
			}
			backoff *= 2
		}

		body, resp, err := f.fetchOnce(ctx, s.URL)
		if err == nil {
			return body, recordOf(resp, body), nil
		}

		lastErr = err
		lastStatus = statusOf(resp)

		if !retryable(lastStatus, err) {
			break
		}
	}

	rec := FetchRecord{URL: s.URL, Status: lastStatus}

	return nil, rec, &FetchError{
		Source:   s.ID,
		URL:      s.URL,
		Attempts: f.opts.Attempts,
		Status:   lastStatus,
		Reason:   lastErr,
	}
}

// response carries the parts of an http.Response that outlive the
// request, so the body is closed before any of it is used.
type response struct {
	status       int
	finalURL     string
	etag         string
	lastModified string
}

// fetchOnce performs a single attempt. Keeping it separate bounds the
// response body's lifetime to this function, so the retry loop never
// holds an open body.
func (f *Fetcher) fetchOnce(ctx context.Context, url string) ([]byte, *response, error) {
	ctx, cancel := context.WithTimeout(ctx, f.opts.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", f.opts.UserAgent)

	resp, err := f.opts.Client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	meta := &response{
		status:       resp.StatusCode,
		finalURL:     url,
		etag:         resp.Header.Get("ETag"),
		lastModified: resp.Header.Get("Last-Modified"),
	}
	if resp.Request != nil && resp.Request.URL != nil {
		meta.finalURL = resp.Request.URL.String()
	}

	if resp.StatusCode != http.StatusOK {
		return nil, meta, fmt.Errorf("%w: %d", ErrBadStatus, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, meta, fmt.Errorf("read body: %w", err)
	}

	return body, meta, nil
}

func recordOf(resp *response, body []byte) FetchRecord {
	sum := sha256.Sum256(body)

	return FetchRecord{
		Bytes:        int64(len(body)),
		ETag:         resp.etag,
		LastModified: resp.lastModified,
		SHA256:       hex.EncodeToString(sum[:]),
		Status:       resp.status,
		URL:          resp.finalURL,
	}
}

func statusOf(resp *response) int {
	if resp == nil {
		return 0
	}

	return resp.status
}

// retryable reports whether another attempt is worthwhile: transport
// errors, rate limiting, and server errors are transient; every other
// status is not.
func retryable(status int, err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if status == 0 {
		return err != nil
	}
	if status == http.StatusTooManyRequests {
		return true
	}

	return status >= http.StatusInternalServerError
}

// sleepCtx waits for d, or returns early if ctx is done.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("backoff: %w", ctx.Err())
	case <-t.C:
		return nil
	}
}

// LoadPages reads back the pages a Fetch wrote, keyed by source id. A
// source whose file is absent is skipped, which is how an optional
// probe that did not answer stays out of the extractor input.
func LoadPages(workDir string, srcs []Source) (map[string][]byte, error) {
	pages := make(map[string][]byte, len(srcs))

	for _, s := range srcs {
		path := filepath.Join(workDir, s.File())

		body, err := os.ReadFile(path)
		switch {
		case err == nil:
			pages[s.ID] = body
		case errors.Is(err, os.ErrNotExist) && s.Optional:
			continue
		default:
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
	}

	return pages, nil
}

// writeJSONFile writes v with the canonical digest encoding, so every
// artifact this package produces is byte-comparable across runs.
func writeJSONFile(path string, v any) error {
	var buf bytes.Buffer
	if err := encodeJSON(&buf, v); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}
