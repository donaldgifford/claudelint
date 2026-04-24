package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/config"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/discovery"
	"github.com/donaldgifford/claudelint/internal/engine"
	"github.com/donaldgifford/claudelint/internal/reporter"
)

// Output-format identifiers for --format. Values are stable — scripts
// and CI workflows pin on them.
const (
	formatText   = "text"
	formatJSON   = "json"
	formatGitHub = "github"
)

// Exit codes returned by `claudelint run` (and the bare alias). The
// contract is documented in README: 0 = clean, 1 = diagnostics
// failed the run, 2 = usage / config / I/O error.
const (
	exitSuccess   = 0
	exitHasErrors = 1
	exitUsage     = 2
)

// errDiagnostics is a sentinel error returned by runLint when the
// combined severity outcome (errors found, or warnings above --max-warnings)
// requires a non-zero exit. main maps it to exitHasErrors without
// printing an "Error: ..." banner — the diagnostics themselves are
// the message.
var errDiagnostics = errors.New("diagnostics failed the run")

// runOptions is the flag state for `claudelint run`. Keeping it in a
// struct makes wiring subcommand flags to the runner straightforward
// and lets tests exercise the same code path main() does.
type runOptions struct {
	configPath  string
	format      string
	noColor     bool
	quiet       bool
	verbose     bool
	maxWarnings int    // -1 means "unlimited" (the default)
	profileDir  string // empty disables profiling
}

// newRunCmd returns the `run` subcommand. It walks every target path,
// parses each discovered artifact, dispatches the rule registry via
// the engine, and prints diagnostics in the selected format.
func newRunCmd() *cobra.Command {
	var opts runOptions
	cmd := &cobra.Command{
		Use:   "run [path...]",
		Short: "Lint Claude artifacts under the given paths (default: cwd)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLint(cmd, opts, args)
		},
	}
	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .claudelint.hcl (default: walk up from cwd)")
	cmd.Flags().StringVar(&opts.format, "format", formatText, "output format: text, json, or github")
	cmd.Flags().BoolVar(&opts.noColor, "no-color", false, "disable ANSI color in text output (also honors NO_COLOR env)")
	cmd.Flags().BoolVar(&opts.quiet, "quiet", false, "print only error-severity diagnostics and the trailing summary")
	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "print the loaded config path and every in-source-suppressed diagnostic")
	cmd.Flags().IntVar(&opts.maxWarnings, "max-warnings", -1, "fail the run if warnings exceed N (default: unlimited)")
	cmd.Flags().StringVar(&opts.profileDir, "profile", "", "write cpu.pprof, heap.pprof, block.pprof, mutex.pprof to this directory")
	return cmd
}

func runLint(cmd *cobra.Command, opts runOptions, args []string) error {
	if err := validateFormat(opts.format); err != nil {
		return err
	}
	if opts.profileDir != "" {
		session, err := startProfileSession(opts.profileDir)
		if err != nil {
			return fmt.Errorf("start profiling: %w", err)
		}
		defer closeProfileSession(cmd.ErrOrStderr(), session)
	}
	return runLintCore(cmd, opts, args)
}

// runLintCore is the profile-free happy path, split out so runLint's
// cyclomatic complexity stays under the project limit.
func runLintCore(cmd *cobra.Command, opts runOptions, args []string) error {
	targets := args
	if len(targets) == 0 {
		targets = []string{"."}
	}

	startDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve cwd: %w", err)
	}
	loadRes, err := config.Load(opts.configPath, startDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	rc := resolveConfig(loadRes)

	if opts.verbose && loadRes != nil {
		if _, werr := fmt.Fprintf(cmd.ErrOrStderr(), "claudelint: using config %s\n", loadRes.Path); werr != nil {
			return fmt.Errorf("write verbose header: %w", werr)
		}
	}

	cands, err := discoverAll(targets)
	if err != nil {
		return err
	}
	cands = filterIgnoredPaths(cands, rc)

	arts, parseErrs := parseAll(cands)
	res := engine.New(rc).Run(arts, parseErrs)

	summary := reporter.Summary{
		Diagnostics: filterByQuiet(res.Diagnostics, opts.quiet),
		Files:       res.Files,
	}
	if err := writeReport(cmd.OutOrStdout(), summary, opts); err != nil {
		return err
	}
	if opts.verbose {
		if err := writeSuppressedVerbose(cmd.ErrOrStderr(), res.Suppressed); err != nil {
			return fmt.Errorf("write suppressed diagnostics: %w", err)
		}
	}
	if runFailed(res.Diagnostics, opts.maxWarnings) {
		return errDiagnostics
	}
	return nil
}

// closeProfileSession flushes the pprof files and reports any failure
// to stderr. Called from defer — we can't return the error, and a
// failed stderr write is terminal anyway, so best-effort is fine.
func closeProfileSession(w io.Writer, s *profileSession) {
	if err := s.Close(); err != nil {
		_, _ = fmt.Fprintf(w, "claudelint: profiling close: %v\n", err) //nolint:errcheck // best-effort defer write
	}
}

// validateFormat keeps the --format flag strict: an unknown value is
// a usage error, not a silent fallback to text.
func validateFormat(f string) error {
	switch f {
	case formatText, formatJSON, formatGitHub:
		return nil
	}
	return fmt.Errorf("unknown --format %q (want text, json, or github)", f)
}

// resolveConfig adapts a LoadResult (possibly nil) into a
// ResolvedConfig. When no config was loaded we still want a usable
// ResolvedConfig so rule defaults apply.
func resolveConfig(lr *config.LoadResult) *config.ResolvedConfig {
	if lr == nil {
		return config.Resolve(nil)
	}
	return config.Resolve(lr.File).WithPath(lr.Path)
}

// filterIgnoredPaths drops candidates that match any top-level
// `ignore.paths` glob in config. Discovery already honors .gitignore;
// this is the config-level counterpart.
func filterIgnoredPaths(cands []discovery.Candidate, rc *config.ResolvedConfig) []discovery.Candidate {
	out := cands[:0]
	for i := range cands {
		if rc.PathIgnored(cands[i].Path) {
			continue
		}
		out = append(out, cands[i])
	}
	return out
}

// discoverAll expands every user-supplied target into Candidates.
// Errors surface with the target path so users can tell which input
// broke discovery.
func discoverAll(targets []string) ([]discovery.Candidate, error) {
	w := discovery.New(discovery.Options{})
	var cands []discovery.Candidate
	for _, t := range targets {
		got, err := w.Walk(t)
		if err != nil {
			return nil, fmt.Errorf("discover %s: %w", t, err)
		}
		cands = append(cands, got...)
	}
	return cands, nil
}

// writeReport dispatches to the right formatter for opts.format.
// Color handling lives here so text goes through
// reporter.TextWithOptions with the resolved color flag.
func writeReport(w io.Writer, s reporter.Summary, opts runOptions) error {
	switch opts.format {
	case formatJSON:
		return reporter.JSON(w, s)
	case formatGitHub:
		return reporter.GitHub(w, s)
	case formatText:
		color := reporter.ShouldUseColor(opts.noColor, isTerminal(w))
		return reporter.TextWithOptions(w, s, reporter.TextOptions{Color: color})
	}
	return fmt.Errorf("unreachable: unknown format %q", opts.format)
}

// isTerminal returns true when w is an *os.File attached to a TTY.
// Non-file writers (buffers, pipes) get false so tests and CI jobs
// see uncolored output by default.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// filterByQuiet drops every non-error diagnostic when --quiet is set.
// --quiet is for CI gates that only care about blocking findings; the
// trailing summary still prints so scripts can confirm the run ran.
func filterByQuiet(ds []diag.Diagnostic, quiet bool) []diag.Diagnostic {
	if !quiet {
		return ds
	}
	out := ds[:0]
	for i := range ds {
		if ds[i].Severity == diag.SeverityError {
			out = append(out, ds[i])
		}
	}
	return out
}

// writeSuppressedVerbose lists every in-source-suppressed diagnostic
// so users can see what --verbose would have surfaced. Config-level
// suppressions prevent the diagnostic from existing at all and so
// aren't represented here; the config path printed at run start is the
// equivalent surfacing for that mechanism.
func writeSuppressedVerbose(w io.Writer, suppressed []engine.SuppressedDiagnostic) error {
	if len(suppressed) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(w, "claudelint: %d diagnostic(s) suppressed by in-source markers:\n", len(suppressed)); err != nil {
		return err
	}
	for i := range suppressed {
		s := &suppressed[i]
		d := &s.Diagnostic
		if d.Range.Start.IsZero() {
			if _, err := fmt.Fprintf(w, "  %s: %s [%s] (%s)\n", d.Path, d.Message, d.RuleID, s.Reason); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "  %s:%d:%d: %s [%s] (%s)\n",
			d.Path, d.Range.Start.Line, d.Range.Start.Column,
			d.Message, d.RuleID, s.Reason,
		); err != nil {
			return err
		}
	}
	return nil
}

// runFailed reports whether the run should exit non-zero. Any error
// diagnostic fails; warnings fail only when they exceed maxWarnings.
// maxWarnings of -1 means "no limit".
func runFailed(ds []diag.Diagnostic, maxWarnings int) bool {
	var errs, warns int
	for i := range ds {
		switch ds[i].Severity {
		case diag.SeverityError:
			errs++
		case diag.SeverityWarning:
			warns++
		}
	}
	if errs > 0 {
		return true
	}
	if maxWarnings >= 0 && warns > maxWarnings {
		return true
	}
	return false
}

// parseAll runs each Candidate through the kind-specific parser and
// returns (successes, parse-errors). Walking and parsing are split so
// discovery stays I/O-bound while the engine can start scheduling
// rules as soon as parsing finishes.
func parseAll(cands []discovery.Candidate) ([]artifact.Artifact, []*artifact.ParseError) {
	arts := make([]artifact.Artifact, 0, len(cands))
	var errs []*artifact.ParseError

	for _, c := range cands {
		src, err := os.ReadFile(c.AbsPath)
		if err != nil {
			errs = append(errs, &artifact.ParseError{
				Path:    c.Path,
				Message: fmt.Sprintf("read file: %s", err.Error()),
				Cause:   err,
				Range:   diag.Range{},
			})
			continue
		}
		a, perr := parseOne(c, src)
		if perr != nil {
			errs = append(errs, perr)
			continue
		}
		arts = append(arts, a)
	}
	return arts, errs
}

// parseOne dispatches to the right parser for c.Kind. Skill
// companions are indexed after a successful parse so rule code has
// them available through the Artifact surface.
func parseOne(c discovery.Candidate, src []byte) (artifact.Artifact, *artifact.ParseError) {
	switch c.Kind {
	case artifact.KindClaudeMD:
		return artifact.ParseClaudeMD(c.Path, src)
	case artifact.KindCommand:
		return artifact.ParseCommand(c.Path, src)
	case artifact.KindAgent:
		return artifact.ParseAgent(c.Path, src)
	case artifact.KindSkill:
		s, perr := artifact.ParseSkill(c.Path, src)
		if perr != nil {
			return nil, perr
		}
		// Best-effort companion indexing; failure here should not
		// kill the parse. The skill directory is the parent of the
		// SKILL.md file.
		if err := artifact.IndexSkillCompanions(s, absSkillDir(c.AbsPath)); err != nil {
			// Indexing failures mean "no companion data" — skill rules
			// still run, they just can't cross-check references.
			_ = err
		}
		return s, nil
	case artifact.KindHook:
		return artifact.ParseHook(c.Path, src)
	case artifact.KindPlugin:
		return artifact.ParsePlugin(c.Path, src)
	case artifact.KindMarketplace:
		return artifact.ParseMarketplace(c.Path, src)
	default:
		return nil, &artifact.ParseError{
			Path:    c.Path,
			Message: fmt.Sprintf("unknown kind %q", c.Kind),
		}
	}
}

func absSkillDir(skillFilePath string) string {
	// SKILL.md is always directly inside the skill directory, so the
	// parent of the file path is what IndexSkillCompanions wants.
	for i := len(skillFilePath) - 1; i >= 0; i-- {
		if skillFilePath[i] == '/' || skillFilePath[i] == '\\' {
			return skillFilePath[:i]
		}
	}
	return "."
}
