package upstream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// The specdrift command tree lives beside the library rather than in
// internal/cli because it ships with neither the linter nor its
// releases. It is a development tool, built only by "go run
// ./cmd/specdrift", so keeping it here also keeps net/http and the
// fetch logic out of every import path that reaches the linter.

// Exit codes, matching claudelint's own convention.
const (
	// ExitClean means the committed digest matches upstream.
	ExitClean = 0
	// ExitDrift means the digest is out of date. It is the only
	// non-zero code a healthy run can produce.
	ExitDrift = 1
	// ExitFailure means the tool could not decide: a source would not
	// fetch, an anchor moved, or a file could not be written.
	ExitFailure = 2
)

// Committed artifact paths, relative to the repository root. They live
// inside the package directory because go:embed cannot reach outside it
// (DESIGN-0006 OQ2).
const (
	// DigestPath is the committed digest.
	DigestPath = "internal/upstream/digest.json"
	// LockPath is the committed source lock.
	LockPath = "internal/upstream/sources.lock.json"
	// DefaultWorkDir is where a pull lands when no directory is given.
	DefaultWorkDir = "build/specdrift"
)

// Report formats.
const (
	formatText     = "text"
	formatJSON     = "json"
	formatMarkdown = "markdown"
)

// ErrDrift is the sentinel behind exit 1. It carries no detail of its
// own: the report has already been written by the time it is returned.
var ErrDrift = errors.New("upstream drift detected")

// ExitCode maps an error from Check or from the command tree onto a
// process exit code.
func ExitCode(err error) int {
	switch {
	case err == nil:
		return ExitClean
	case errors.Is(err, ErrDrift):
		return ExitDrift
	default:
		return ExitFailure
	}
}

// Execute runs the specdrift command tree and returns a process exit
// code.
func Execute(args []string, stdout, stderr io.Writer) int {
	root := NewRootCommand()
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)

	err := root.Execute()
	code := ExitCode(err)

	if code == ExitFailure {
		if _, werr := fmt.Fprintln(stderr, "specdrift:", err); werr != nil {
			// Nothing useful is left to do if stderr is gone.
			return ExitFailure
		}
	}

	return code
}

// NewRootCommand assembles the specdrift command tree.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "specdrift",
		Short: "Detect drift between claudelint's ruleset and the upstream Claude Code specs",
		Long: "specdrift fetches the published Claude Code and Agent Skills sources, " +
			"extracts a deterministic digest of the facts claudelint's rules encode, " +
			"and diffs it against the digest committed in this repository. " +
			"Exit 0 means no drift, 1 means the digest is out of date, and 2 means " +
			"the tool could not decide.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newPullCommand(), newDigestCommand(), newDiffCommand(), newCheckCommand())

	return root
}

// newPullCommand fetches every source into a work directory.
func newPullCommand() *cobra.Command {
	var out string

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Fetch every upstream source into a work directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manifest, err := pull(cmd.Context(), out)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "fetched %d sources into %s\n", len(manifest), out)

			return err
		},
	}
	cmd.Flags().StringVar(&out, "out", DefaultWorkDir, "directory to fetch into")

	return cmd
}

// newDigestCommand extracts a digest from an already-fetched work
// directory.
func newDigestCommand() *cobra.Command {
	var in, out, lock, snippets string

	cmd := &cobra.Command{
		Use:   "digest",
		Short: "Extract a digest from a work directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDigest(cmd.OutOrStdout(), in, out, lock, snippets)
		},
	}
	cmd.Flags().StringVar(&in, "in", DefaultWorkDir, "work directory holding fetched sources")
	cmd.Flags().StringVar(&out, "out", DigestPath, "digest file to write")
	cmd.Flags().StringVar(&lock, "lock", LockPath, "lock file to write")
	cmd.Flags().StringVar(&snippets, "write-snippets", "",
		"also write the golden test fixtures into this directory")

	return cmd
}

// newDiffCommand compares two digest files.
func newDiffCommand() *cobra.Command {
	var base, head, format, out string

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Structurally diff two digest files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			report, err := diffFiles(base, head)
			if err != nil {
				return err
			}

			if err := emit(cmd.OutOrStdout(), report, format, out); err != nil {
				return err
			}

			if report.HasDrift() {
				return ErrDrift
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&base, "base", DigestPath, "digest to compare against")
	cmd.Flags().StringVar(&head, "head", "", "digest to compare (required)")
	cmd.Flags().StringVar(&format, "format", formatText, "text, json, or markdown")
	cmd.Flags().StringVar(&out, "out", "", "write the report to this file instead of stdout")
	must(cmd.MarkFlagRequired("head"))

	return cmd
}

// CheckOptions configures a Check run. The cobra command fills it from
// flags; a test fills it directly, which is how the pipeline is
// exercised end to end without touching the network.
type CheckOptions struct {
	// Work is where the fetched sources land. An empty Work uses a
	// temporary directory and removes it afterwards.
	Work string
	// Format is the report format: text, json, or markdown.
	Format string
	// Out is a file to write the report to instead of stdout.
	Out string
	// Digest is the committed digest to compare against.
	Digest string
	// Lock is the committed lock to compare against.
	Lock string
	// Update rewrites the committed digest and lock instead of
	// reporting.
	Update bool
	// Snippets, when set, is where the golden fixtures are written.
	Snippets string
	// JSON, when set, is a file the report is also written to as JSON,
	// whatever Format is. The workflow needs the Markdown report for the
	// issue body and the JSON for the change marker, from one fetch.
	JSON string
	// Fetch tunes the HTTP client. The zero value is the production
	// configuration.
	Fetch FetchOptions
}

// newCheckCommand is the CI entry point: pull, digest, diff.
func newCheckCommand() *cobra.Command {
	var opts CheckOptions

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Fetch, extract, and diff against the committed digest",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return Check(cmd.Context(), cmd.OutOrStdout(), &opts)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.Work, "work", "",
		"keep the fetched sources in this directory instead of a temporary one")
	f.StringVar(&opts.Format, "format", formatText, "text, json, or markdown")
	f.StringVar(&opts.Out, "out", "", "write the report to this file instead of stdout")
	f.StringVar(&opts.Digest, "digest", DigestPath, "committed digest to compare against")
	f.StringVar(&opts.Lock, "lock", LockPath, "committed lock to compare against")
	f.BoolVar(&opts.Update, "update", false, "rewrite the committed digest and lock instead of reporting")
	f.StringVar(&opts.Snippets, "write-snippets", "",
		"also write the golden test fixtures into this directory")
	f.StringVar(&opts.JSON, "json", "", "also write the report as JSON to this file")

	return cmd
}

// pull fetches every source into workDir.
func pull(ctx context.Context, workDir string) (Manifest, error) {
	return NewFetcher(FetchOptions{}).Fetch(ctx, Sources(), workDir)
}

// runDigest extracts from a work directory and writes the digest and
// lock.
func runDigest(stdout io.Writer, workDir, digestPath, lockPath, snippetDir string) error {
	manifest, err := LoadManifest(workDir)
	if err != nil {
		return err
	}

	pages, err := LoadPages(workDir, Sources())
	if err != nil {
		return err
	}

	digest, lock, err := extractAndLock(pages, manifest, digestPath, snippetDir)
	if err != nil {
		return err
	}

	if err := writeArtifacts(digest, lock, digestPath, lockPath); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "wrote %s and %s\n", digestPath, lockPath)

	return err
}

// extractAndLock runs the extractors and builds the lock. The committed
// digest, when there is one, supplies the per-section sanity floors.
func extractAndLock(
	pages map[string][]byte,
	manifest Manifest,
	baselinePath, snippetDir string,
) (*Digest, Lock, error) {
	baseline, found, err := loadBaseline(baselinePath)
	if err != nil {
		return nil, nil, err
	}

	opts := ExtractOptions{}
	if found {
		opts.Baseline = baseline
	}
	if snippetDir != "" {
		opts.Snippets = make(map[string]string)
	}

	digest, err := Extract(pages, opts)
	if err != nil {
		return nil, nil, err
	}

	if snippetDir != "" {
		if err := WriteSnippets(snippetDir, opts.Snippets); err != nil {
			return nil, nil, err
		}
	}

	return digest, NewLock(manifest, pages), nil
}

// Check is the whole pipeline: fetch, extract, then either rewrite the
// committed files or report against them. It returns ErrDrift when the
// committed digest is out of date, which ExitCode turns into exit 1.
func Check(ctx context.Context, stdout io.Writer, opts *CheckOptions) error {
	workDir, cleanup, err := checkWorkDir(opts.Work)
	if err != nil {
		return err
	}
	defer cleanup()

	manifest, err := NewFetcher(opts.Fetch).Fetch(ctx, Sources(), workDir)
	if err != nil {
		return err
	}

	pages, err := LoadPages(workDir, Sources())
	if err != nil {
		return err
	}

	head, headLock, err := extractAndLock(pages, manifest, opts.Digest, opts.Snippets)
	if err != nil {
		return err
	}

	if opts.Update {
		if err := writeArtifacts(head, headLock, opts.Digest, opts.Lock); err != nil {
			return err
		}

		_, err = fmt.Fprintf(stdout, "updated %s and %s\n", opts.Digest, opts.Lock)

		return err
	}

	report, err := checkReport(head, headLock, opts)
	if err != nil {
		return err
	}

	if opts.JSON != "" {
		raw, err := report.JSON()
		if err != nil {
			return err
		}
		if err := writeFile(opts.JSON, raw); err != nil {
			return err
		}
	}

	if err := emit(stdout, report, opts.Format, opts.Out); err != nil {
		return err
	}

	if report.HasDrift() {
		return ErrDrift
	}

	return nil
}

// checkReport diffs the freshly extracted digest against the committed
// one and attaches the lock diff.
func checkReport(head *Digest, headLock Lock, opts *CheckOptions) (*Report, error) {
	committed, err := LoadDigest(opts.Digest)
	if err != nil {
		return nil, err
	}

	report, err := Diff(committed, head)
	if err != nil {
		return nil, err
	}

	committedLock, err := LoadLock(opts.Lock)
	if err != nil {
		return nil, err
	}
	report.SourcesChanged = ChangedSources(committedLock, headLock)

	return report, nil
}

// checkWorkDir returns the directory to fetch into and a cleanup for
// it. A --work directory is kept, because that is the point of the
// flag: the workflow uploads it as an artifact when a run fails.
func checkWorkDir(work string) (string, func(), error) {
	if work != "" {
		return work, func() {}, nil
	}

	dir, err := os.MkdirTemp("", "specdrift-")
	if err != nil {
		return "", nil, fmt.Errorf("create work directory: %w", err)
	}

	// A failure to remove a temporary directory is not worth failing a
	// run that has already produced its report.
	cleanup := func() {
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			fmt.Fprintln(os.Stderr, "specdrift: remove work directory:", rmErr)
		}
	}

	return dir, cleanup, nil
}

// diffFiles compares two digest files.
func diffFiles(basePath, headPath string) (*Report, error) {
	base, err := LoadDigest(basePath)
	if err != nil {
		return nil, err
	}

	head, err := LoadDigest(headPath)
	if err != nil {
		return nil, err
	}

	return Diff(base, head)
}

// LoadDigest reads and parses a digest file.
func LoadDigest(path string) (*Digest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return Decode(raw)
}

// loadBaseline reads a digest, reporting a missing file as absent
// rather than as an error. Bootstrapping the very first digest has
// nothing to compare against, and the sanity floors have to tolerate
// that.
func loadBaseline(path string) (digest *Digest, found bool, err error) {
	digest, err = LoadDigest(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, false, nil
	case err != nil:
		return nil, false, err
	default:
		return digest, true, nil
	}
}

// writeArtifacts writes the digest and the lock.
func writeArtifacts(digest *Digest, lock Lock, digestPath, lockPath string) error {
	raw, err := digest.Encode()
	if err != nil {
		return err
	}
	if err := writeFile(digestPath, raw); err != nil {
		return err
	}

	rawLock, err := lock.Encode()
	if err != nil {
		return err
	}

	return writeFile(lockPath, rawLock)
}

// emit renders a report to a file or to stdout.
func emit(stdout io.Writer, report *Report, format, out string) error {
	rendered, err := render(report, format)
	if err != nil {
		return err
	}

	if out == "" {
		_, err := stdout.Write(rendered)

		return err
	}

	if err := writeFile(out, rendered); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "%s\nreport written to %s\n", summary(report), out)

	return err
}

// render turns a report into bytes in the requested format.
func render(report *Report, format string) ([]byte, error) {
	switch format {
	case formatText:
		return []byte(report.Text()), nil
	case formatJSON:
		return report.JSON()
	case formatMarkdown:
		return []byte(report.Markdown()), nil
	default:
		return nil, fmt.Errorf("unknown format %q: want %s, %s, or %s",
			format, formatText, formatJSON, formatMarkdown)
	}
}

// summary is the one-line outcome printed when the report itself went
// to a file.
func summary(report *Report) string {
	if !report.HasDrift() {
		return "no drift"
	}

	return fmt.Sprintf("drift: %d changes", len(report.Changes))
}

// writeFile writes b to path, creating the parent directory.
func writeFile(path string, b []byte) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// must panics on a flag-wiring error, which can only be a programming
// mistake in this file.
func must(err error) {
	if err != nil {
		panic(err)
	}
}
