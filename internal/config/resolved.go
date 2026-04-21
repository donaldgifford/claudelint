package config

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/donaldgifford/claudelint/internal/diag"
)

// ResolvedConfig is the engine's view of user configuration. It is
// built from a *File once at startup so per-artifact lookups run in
// O(1) for rule enablement, severity, and option values; per-path
// suppression is linear in the number of configured globs (small in
// practice, and the engine can cache matchers further if needed).
//
// A zero ResolvedConfig is a valid "no user config" state: every rule
// is enabled at its default severity, no paths are ignored, and
// options come from Rule.DefaultOptions().
type ResolvedConfig struct {
	path              string
	output            string
	ignorePaths       []string
	ruleEnabled       map[string]bool
	ruleSeverity      map[string]diag.Severity
	ruleOptions       map[string]map[string]any
	rulePaths         map[string][]string // per-rule path-suppression globs
	kindSeverity      map[string]diag.Severity
	kindOptions       map[string]map[string]any
	configuredRuleIDs []string // sorted, deduped IDs that appear in rule "<id>" blocks
}

// Resolve builds a ResolvedConfig from a *File. It is safe to pass a
// nil *File — the result is the zero-value resolved config.
func Resolve(f *File) *ResolvedConfig {
	rc := &ResolvedConfig{
		ruleEnabled:  make(map[string]bool),
		ruleSeverity: make(map[string]diag.Severity),
		ruleOptions:  make(map[string]map[string]any),
		rulePaths:    make(map[string][]string),
		kindSeverity: make(map[string]diag.Severity),
		kindOptions:  make(map[string]map[string]any),
	}
	if f == nil {
		return rc
	}

	if f.Output != nil {
		rc.output = f.Output.Format
	}
	if f.Ignore != nil {
		rc.ignorePaths = append(rc.ignorePaths, f.Ignore.Paths...)
	}
	for _, rk := range f.RulesKind {
		if rk.DefaultSeverity != "" {
			// Severity strings are validated before Resolve is called,
			// so the parse cannot fail here.
			rc.kindSeverity[rk.Kind] = mustSeverity(rk.DefaultSeverity)
		}
		if !rk.Options.IsNull() {
			rc.kindOptions[rk.Kind] = ctyValueToMap(rk.Options)
		}
	}
	seen := make(map[string]struct{}, len(f.Rules))
	for _, r := range f.Rules {
		if _, ok := seen[r.ID]; !ok {
			seen[r.ID] = struct{}{}
			rc.configuredRuleIDs = append(rc.configuredRuleIDs, r.ID)
		}
		if r.Enabled != nil {
			rc.ruleEnabled[r.ID] = *r.Enabled
		}
		if r.Severity != "" {
			rc.ruleSeverity[r.ID] = mustSeverity(r.Severity)
		}
		if !r.Options.IsNull() {
			rc.ruleOptions[r.ID] = ctyValueToMap(r.Options)
		}
		if len(r.Paths) > 0 {
			rc.rulePaths[r.ID] = append(rc.rulePaths[r.ID], r.Paths...)
		}
	}
	sort.Strings(rc.configuredRuleIDs)
	return rc
}

// WithPath records the absolute path of the .claudelint.hcl that
// produced this ResolvedConfig. Returns the receiver so the CLI can
// chain `config.Resolve(lr.File).WithPath(lr.Path)`. When no config
// was loaded, callers should leave the path empty.
func (rc *ResolvedConfig) WithPath(p string) *ResolvedConfig {
	rc.path = p
	return rc
}

// Path returns the absolute path of the config file that produced this
// ResolvedConfig, or "" when no config file was loaded. The engine uses
// it as the Path field on meta/unknown-rule diagnostics so users can
// jump straight to the misspelled ID.
func (rc *ResolvedConfig) Path() string { return rc.path }

// ConfiguredRuleIDs returns every rule ID that appears in a `rule "<id>"`
// block in the loaded config, sorted and deduplicated. The engine
// cross-checks these against the registry to emit meta/unknown-rule
// warnings for typos.
func (rc *ResolvedConfig) ConfiguredRuleIDs() []string {
	return rc.configuredRuleIDs
}

// RuleEnabled reports whether the rule identified by id is enabled.
// The default is true: users opt rules out rather than in.
func (rc *ResolvedConfig) RuleEnabled(id string) bool {
	if v, ok := rc.ruleEnabled[id]; ok {
		return v
	}
	return true
}

// RuleSeverity returns the configured severity for id, or defaultSev
// if the user has not set one.
//
// Resolution order: per-rule severity wins; per-kind default is a
// fallback; finally defaultSev (the rule's DefaultSeverity()) applies.
func (rc *ResolvedConfig) RuleSeverity(id, kind string, defaultSev diag.Severity) diag.Severity {
	if v, ok := rc.ruleSeverity[id]; ok {
		return v
	}
	if v, ok := rc.kindSeverity[kind]; ok {
		return v
	}
	return defaultSev
}

// RuleOption returns the option value for (id, key), falling back to
// the per-kind default and finally the provided default value. The
// engine calls this after overlaying the rule's DefaultOptions on top
// of the per-rule block at load time; callers get a single, resolved
// value without having to juggle layers.
func (rc *ResolvedConfig) RuleOption(id, kind, key string, def any) any {
	if opts, ok := rc.ruleOptions[id]; ok {
		if v, exists := opts[key]; exists {
			return v
		}
	}
	if opts, ok := rc.kindOptions[kind]; ok {
		if v, exists := opts[key]; exists {
			return v
		}
	}
	return def
}

// PathIgnored reports whether repo-relative path p is suppressed by
// any top-level `ignore.paths` glob. Per-rule `paths` globs are
// checked separately via PathIgnoredForRule.
func (rc *ResolvedConfig) PathIgnored(p string) bool {
	return matchAny(rc.ignorePaths, p)
}

// PathIgnoredForRule reports whether a specific rule is suppressed
// for path p via its own `paths` globs.
func (rc *ResolvedConfig) PathIgnoredForRule(id, p string) bool {
	return matchAny(rc.rulePaths[id], p)
}

// Output returns the configured output format, or "" when the user
// did not set one. The engine defaults to "text" in that case.
func (rc *ResolvedConfig) Output() string { return rc.output }

// matchAny reports whether any glob in globs matches path p using
// filepath.Match after normalizing p to slashes. `**` is expanded
// below to support the common prefix/**, **/suffix, and **/mid/**
// patterns gitignore-style tooling exposes.
func matchAny(globs []string, p string) bool {
	if len(globs) == 0 {
		return false
	}
	p = filepath.ToSlash(p)
	for _, g := range globs {
		if strings.Contains(g, "**") {
			if doubleStarMatch(g, p) {
				return true
			}
			continue
		}
		ok, err := filepath.Match(g, p)
		if err == nil && ok {
			return true
		}
	}
	return false
}

// mustSeverity decodes a validated severity string. Callers must have
// already run validateSeverity; an unknown value panics so a bug in
// that validation surfaces loudly instead of silently downgrading.
func mustSeverity(s string) diag.Severity {
	var sev diag.Severity
	if err := sev.UnmarshalText([]byte(s)); err != nil {
		panic("config: unvalidated severity " + s)
	}
	return sev
}

// doubleStarMatch is a minimal gitignore-style `**` matcher. It
// splits the glob on every `**` and requires each segment to appear,
// in order, in the path. The leading segment must start at position
// zero; the trailing segment must reach the end. Middle segments only
// need to exist somewhere. Leading/trailing slashes on segments are
// stripped so `**/testdata/**` matches both `testdata/x` and
// `a/b/testdata/x`.
func doubleStarMatch(glob, p string) bool {
	parts := strings.Split(glob, "**")
	// `a**b` → ["a", "b"]; `**/x/**` → ["", "/x/", ""].
	remaining := p
	for i, part := range parts {
		trimmed := strings.Trim(part, "/")
		switch {
		case i == 0:
			// Leading anchor — must prefix the path.
			if trimmed == "" {
				continue
			}
			if !strings.HasPrefix(remaining, trimmed) {
				return false
			}
			remaining = remaining[len(trimmed):]
			remaining = strings.TrimPrefix(remaining, "/")
		case i == len(parts)-1:
			// Trailing anchor — must suffix what's left.
			if trimmed == "" {
				return true
			}
			return strings.HasSuffix(remaining, trimmed)
		default:
			idx := strings.Index(remaining, trimmed)
			if idx < 0 {
				return false
			}
			remaining = remaining[idx+len(trimmed):]
			remaining = strings.TrimPrefix(remaining, "/")
		}
	}
	return true
}
