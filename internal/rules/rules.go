// Package rules defines the Rule interface, the claudelint registry,
// and the ruleset's version metadata. Rule implementations live in
// subpackages of internal/rules — e.g. internal/rules/skills — and
// register themselves from init() via Register.
//
// The package is deliberately narrow so rule authors only depend on
// it (plus internal/artifact and internal/diag). Engine wiring lives
// in internal/engine and is invisible to rule code.
package rules

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
)

// Rule is the unit of analysis. Rules are:
//
//   - pure — no I/O, no global state, no cross-file awareness in v1.
//   - small — a typical rule is ≤ 50 LOC plus a table-driven test.
//   - focused — one rule checks one property.
//
// The contract is intentionally modelled on go/analysis.Analyzer and
// golangci-lint's linter interface. Engine additions (suppressions,
// concurrency, reporting) never require rule changes.
type Rule interface {
	// ID is the stable identifier used by configuration, CLI output,
	// and in-source suppressions. It follows the "category/name"
	// convention — e.g. "skills/body-size".
	ID() string

	// Category groups rules for `claudelint rules` output and for
	// suppression globs. Canonical values: "schema", "content",
	// "security", "style", "meta".
	Category() string

	// DefaultSeverity is the severity emitted when the user has not
	// overridden it via config.
	DefaultSeverity() diag.Severity

	// DefaultOptions declares the rule's option keys and their
	// default values. The engine fills in unspecified options from
	// this map and uses the Go types of the default values to
	// validate user-supplied options before Check is called.
	// Return nil if the rule takes no options.
	DefaultOptions() map[string]any

	// AppliesTo lists the artifact kinds this rule analyzes. Rules
	// that cover every kind (e.g. schema/parse) can list AllKinds.
	AppliesTo() []artifact.ArtifactKind

	// Check produces zero or more Diagnostics for a single parsed
	// artifact. It is called at most once per (artifact, rule) pair.
	// Implementations must be deterministic and must not rely on
	// cross-file state.
	Check(ctx Context, a artifact.Artifact) []diag.Diagnostic
}

// Context is everything a rule is allowed to see beyond the artifact:
// resolved options for this rule, the rule's own ID, and a leveled
// logger. Kept deliberately narrow so rules stay testable in isolation
// (engine.NewContext for production; testing code can stub the
// interface directly).
type Context interface {
	// RuleID is the ID the engine dispatched this Check under. Rules
	// should use it when constructing diagnostics rather than
	// hardcoding a second copy.
	RuleID() string

	// Option returns the resolved option value for key, falling back
	// to the rule's DefaultOptions value. Option types are enforced
	// by the engine before Check is called — rules can type-assert
	// with confidence.
	Option(key string) any

	// Logf writes a debug-level message to the engine's logger. It
	// is free to drop messages; rules must not rely on log output
	// for correctness.
	Logf(format string, args ...any)
}
