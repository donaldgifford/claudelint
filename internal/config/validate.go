package config

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/diag"
)

// validate checks a decoded *File for schema-version compatibility
// and for semantic errors gohcl cannot catch itself (e.g. a severity
// value that is not in the enum). It returns the first violation as
// a plain error so users see the offending rule ID or block in the
// CLI output.
//
// Validation is intentionally conservative — we prefer rejecting at
// load time over silently downgrading rules, because a misspelled
// severity that becomes "info" would mask real regressions.
func validate(f *File) error {
	if f.Claudelint == nil {
		return fmt.Errorf("missing required `claudelint { version = ... }` block")
	}
	if f.Claudelint.Version != SchemaVersion {
		return fmt.Errorf(
			"unsupported config schema version %q (want %q); upgrade claudelint",
			f.Claudelint.Version, SchemaVersion,
		)
	}

	for _, rk := range f.RulesKind {
		if rk.DefaultSeverity != "" {
			if err := validateSeverity(rk.DefaultSeverity); err != nil {
				return fmt.Errorf(`rules %q: %w`, rk.Kind, err)
			}
		}
	}

	for _, r := range f.Rules {
		if r.Severity != "" {
			if err := validateSeverity(r.Severity); err != nil {
				return fmt.Errorf(`rule %q: %w`, r.ID, err)
			}
		}
		if !r.Options.IsNull() && !r.Options.Type().IsObjectType() && !r.Options.Type().IsMapType() {
			return fmt.Errorf(`rule %q: options must be an object, got %s`, r.ID, r.Options.Type().FriendlyName())
		}
	}

	if f.Output != nil && f.Output.Format != "" {
		if err := validateFormat(f.Output.Format); err != nil {
			return err
		}
	}
	return nil
}

// validateSeverity wraps diag.Severity.UnmarshalText so the error
// message includes the canonical enum values. The caller prepends the
// rule or kind label.
func validateSeverity(s string) error {
	var sev diag.Severity
	if err := sev.UnmarshalText([]byte(s)); err != nil {
		return fmt.Errorf("severity: %w", err)
	}
	return nil
}

// validateFormat rejects unknown output formats at load time so the
// engine never has to fall back to a default silently.
func validateFormat(s string) error {
	switch s {
	case "text", "json", "github":
		return nil
	default:
		return fmt.Errorf(`output.format %q: want one of "text", "json", "github"`, s)
	}
}
