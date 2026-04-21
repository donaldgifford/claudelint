package engine

import (
	"runtime"
	"sort"
	"sync"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/config"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// ParseSchemaRuleID is the rule identifier the engine uses when
// synthesizing a Diagnostic from a *artifact.ParseError. Phase 1.5
// registers a pseudo-rule under this same ID so users can disable
// or re-severity parse errors via the same config surface as any
// other rule.
const ParseSchemaRuleID = "schema/parse"

// Result is the runner's output: the aggregated Diagnostic list plus
// the number of artifacts that were inspected. Reporter.Summary maps
// 1:1 onto this shape.
type Result struct {
	Diagnostics []diag.Diagnostic
	Files       int
}

// Runner orchestrates rule execution. Construct one per invocation;
// the runner owns no long-lived state so concurrent Run calls on
// separate Runners are independent.
type Runner struct {
	cfg     *config.ResolvedConfig
	workers int
	logger  Logger
}

// Option configures a Runner. Functional options let callers override
// defaults without breaking the New signature as the engine grows.
type Option func(*Runner)

// WithWorkers overrides the worker-pool size. The default is
// GOMAXPROCS, which matches the per-artifact granularity decision in
// DESIGN-0001.
func WithWorkers(n int) Option {
	return func(r *Runner) {
		if n > 0 {
			r.workers = n
		}
	}
}

// WithLogger overrides the Logger rules see via Context.Logf. Defaults
// to a Logger that discards messages.
func WithLogger(l Logger) Option {
	return func(r *Runner) { r.logger = l }
}

// New returns a Runner configured with cfg. A nil cfg is treated as
// "no user config" (every rule enabled at its default severity, no
// path ignores).
func New(cfg *config.ResolvedConfig, opts ...Option) *Runner {
	r := &Runner{
		cfg:     cfg,
		workers: runtime.GOMAXPROCS(0),
		logger:  NopLogger(),
	}
	if r.cfg == nil {
		r.cfg = config.Resolve(nil)
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run executes every enabled rule against every artifact and returns
// the aggregated Result. parseErrs synthesizes into schema/parse
// diagnostics without any rule.Check being called.
func (r *Runner) Run(arts []artifact.Artifact, parseErrs []*artifact.ParseError) *Result {
	diags := make([]diag.Diagnostic, 0, 2*len(arts))
	for _, pe := range parseErrs {
		diags = append(diags, r.synthesizeParseDiagnostic(pe))
	}

	if len(arts) > 0 {
		diags = append(diags, r.runRules(arts)...)
	}

	sortAndDedupe(&diags)
	return &Result{Diagnostics: diags, Files: len(arts) + len(parseErrs)}
}

// runRules dispatches rule.Check across arts with a bounded
// goroutine pool. One goroutine handles one artifact; rules that
// apply run serially within that goroutine. See DESIGN-0001 §execution-flow
// for the rationale behind per-artifact granularity.
func (r *Runner) runRules(arts []artifact.Artifact) []diag.Diagnostic {
	applicable := r.applicableRules()
	if len(applicable) == 0 {
		return nil
	}

	type result struct {
		path  string
		diags []diag.Diagnostic
	}

	jobs := make(chan artifact.Artifact, len(arts))
	out := make(chan result, len(arts))
	var wg sync.WaitGroup
	for i := 0; i < r.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for a := range jobs {
				out <- result{path: a.Path(), diags: r.runOne(a, applicable)}
			}
		}()
	}
	for _, a := range arts {
		jobs <- a
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(out)
	}()

	var collected []diag.Diagnostic
	for r := range out {
		collected = append(collected, r.diags...)
	}
	return collected
}

// runOne evaluates every applicable rule against a single artifact.
// applicable is already filtered to rules whose AppliesTo includes
// a.Kind() and whose ID is enabled in config.
func (r *Runner) runOne(a artifact.Artifact, applicable []rules.Rule) []diag.Diagnostic {
	var out []diag.Diagnostic
	for _, rule := range applicable {
		if !ruleAppliesToKind(rule, a.Kind()) {
			continue
		}
		if r.cfg.PathIgnoredForRule(rule.ID(), a.Path()) {
			continue
		}
		options := resolveOptions(rule, a.Kind(), r.cfg)
		c := &ctx{ruleID: rule.ID(), options: options, logger: r.logger}
		diags := rule.Check(c, a)
		for i := range diags {
			r.finalizeDiagnostic(&diags[i], rule, a.Kind())
		}
		out = append(out, diags...)
	}
	return out
}

// applicableRules returns the enabled, non-path-ignored rules in
// stable order. Path-per-rule suppression is still applied per
// artifact inside runOne.
func (r *Runner) applicableRules() []rules.Rule {
	all := rules.All()
	out := make([]rules.Rule, 0, len(all))
	for _, rule := range all {
		if rule.ID() == ParseSchemaRuleID {
			// Pseudo-rule: engine synthesizes the diagnostic, Check
			// is never called.
			continue
		}
		if !r.cfg.RuleEnabled(rule.ID()) {
			continue
		}
		out = append(out, rule)
	}
	return out
}

// finalizeDiagnostic applies config-level severity overrides and
// fills in the rule ID if the rule forgot to. Engine-level
// bookkeeping happens here so rule authors don't have to remember it.
func (r *Runner) finalizeDiagnostic(d *diag.Diagnostic, rule rules.Rule, kind artifact.ArtifactKind) {
	if d.RuleID == "" {
		d.RuleID = rule.ID()
	}
	d.Severity = r.cfg.RuleSeverity(rule.ID(), string(kind), rule.DefaultSeverity())
}

// synthesizeParseDiagnostic builds a schema/parse diagnostic from a
// ParseError. The severity follows config overrides just like any
// regular rule so users can downgrade parse errors to warnings if
// they want the rest of the ruleset to keep running on broken input.
func (r *Runner) synthesizeParseDiagnostic(pe *artifact.ParseError) diag.Diagnostic {
	sev := r.cfg.RuleSeverity(ParseSchemaRuleID, "", diag.SeverityError)
	return diag.Diagnostic{
		RuleID:   ParseSchemaRuleID,
		Severity: sev,
		Path:     pe.Path,
		Range:    pe.Range,
		Message:  pe.Message,
	}
}

// ruleAppliesToKind reports whether a rule's AppliesTo list contains k.
func ruleAppliesToKind(rule rules.Rule, k artifact.ArtifactKind) bool {
	for _, want := range rule.AppliesTo() {
		if want == k {
			return true
		}
	}
	return false
}

// resolveOptions overlays DefaultOptions with per-kind and per-rule
// config overrides. The returned map is freshly allocated so concurrent
// Check calls do not share mutable state.
func resolveOptions(rule rules.Rule, kind artifact.ArtifactKind, cfg *config.ResolvedConfig) map[string]any {
	defaults := rule.DefaultOptions()
	out := make(map[string]any, len(defaults))
	for k, v := range defaults {
		out[k] = cfg.RuleOption(rule.ID(), string(kind), k, v)
	}
	return out
}

// sortAndDedupe sorts diagnostics by (path, line, col, rule) and
// removes exact duplicates. Dedup is O(n) after sort.
func sortAndDedupe(diags *[]diag.Diagnostic) {
	sort.SliceStable(*diags, func(i, j int) bool {
		a, b := (*diags)[i], (*diags)[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Range.Start.Line != b.Range.Start.Line {
			return a.Range.Start.Line < b.Range.Start.Line
		}
		if a.Range.Start.Column != b.Range.Start.Column {
			return a.Range.Start.Column < b.Range.Start.Column
		}
		return a.RuleID < b.RuleID
	})
	if len(*diags) < 2 {
		return
	}
	out := (*diags)[:1]
	for i := 1; i < len(*diags); i++ {
		if (*diags)[i] == out[len(out)-1] {
			continue
		}
		out = append(out, (*diags)[i])
	}
	*diags = out
}
