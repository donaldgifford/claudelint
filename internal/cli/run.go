package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/config"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/discovery"
	"github.com/donaldgifford/claudelint/internal/engine"
	"github.com/donaldgifford/claudelint/internal/reporter"
)

// runOptions is the flag state for `claudelint run`. Keeping it in a
// struct makes wiring subcommand flags to the runner straightforward
// and lets tests exercise the same code path main() does.
type runOptions struct {
	configPath string
}

// newRunCmd returns the `run` subcommand. It walks every target path,
// parses each discovered artifact, dispatches the rule registry via
// the engine, and prints the text-format summary.
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
	return cmd
}

func runLint(cmd *cobra.Command, opts runOptions, args []string) error {
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
	var rc *config.ResolvedConfig
	if loadRes != nil {
		rc = config.Resolve(loadRes.File)
	} else {
		rc = config.Resolve(nil)
	}

	w := discovery.New(discovery.Options{})
	var cands []discovery.Candidate
	for _, t := range targets {
		got, err := w.Walk(t)
		if err != nil {
			return fmt.Errorf("discover %s: %w", t, err)
		}
		cands = append(cands, got...)
	}

	arts, parseErrs := parseAll(cands)
	res := engine.New(rc).Run(arts, parseErrs)
	return reporter.Text(cmd.OutOrStdout(), reporter.Summary{
		Diagnostics: res.Diagnostics,
		Files:       res.Files,
	})
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
