// Package engine is claudelint's runner. It wires discovery + parsing
// output into the rule registry, dispatches rule.Check with a bounded
// worker pool, synthesizes schema/parse diagnostics from parse
// errors, and aggregates output for the reporter.
//
// Rules do not import engine; engine imports rules. This package is
// where all orchestration complexity lives so every rule can remain a
// tiny pure Check function.
package engine

import (
	"io"
	"log"

	"github.com/donaldgifford/claudelint/internal/rules"
)

// Logger is the minimal surface rules see through Context.Logf. The
// engine feeds it a *log.Logger in production and can swap in a
// nop-logger during tests to keep output clean.
type Logger interface {
	Logf(format string, args ...any)
}

// ctx is the engine's implementation of rules.Context. It is
// constructed per-(rule,artifact) pair so Option lookups are trivial
// and there is no shared-mutable state between concurrent Check
// invocations.
type ctx struct {
	ruleID  string
	options map[string]any
	logger  Logger
}

// RuleID implements rules.Context.
func (c *ctx) RuleID() string { return c.ruleID }

// Option implements rules.Context. Options are pre-resolved by the
// engine: the map already layers DefaultOptions, per-kind config, and
// per-rule config in the correct precedence order.
func (c *ctx) Option(key string) any { return c.options[key] }

// Logf implements rules.Context.
func (c *ctx) Logf(format string, args ...any) {
	if c.logger == nil {
		return
	}
	c.logger.Logf(format, args...)
}

// Compile-time proof.
var _ rules.Context = (*ctx)(nil)

// stdLogger adapts the stdlib *log.Logger to our Logger interface.
type stdLogger struct{ l *log.Logger }

// Logf implements Logger.
func (s *stdLogger) Logf(format string, args ...any) {
	s.l.Printf(format, args...)
}

// NopLogger returns a Logger that discards every message. Useful for
// tests and for runs where the rule-level log stream would be noisy.
func NopLogger() Logger { return &stdLogger{l: log.New(io.Discard, "", 0)} }
