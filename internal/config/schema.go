// Package config loads .claudelint.hcl (schema v1), validates it, and
// produces a ResolvedConfig the engine consumes. The HCL-level structs
// below mirror the on-disk schema verbatim; ResolvedConfig collapses
// them into O(1) lookups so the hot path never walks the raw config.
//
// Dependency direction: config knows about diag (for severities) but
// nothing about rules or artifacts. Phase 1.5 rules read their options
// through the ResolvedConfig.RuleOption accessor — they never touch
// the HCL AST.
package config

import "github.com/zclconf/go-cty/cty"

// SchemaVersion is the only schema string this binary accepts. A
// future breaking change bumps this to "2" and the loader rejects
// "1" unless users explicitly upgrade their file.
const SchemaVersion = "1"

// File is the top-level HCL document. Each field maps to a named
// block (or list of blocks) defined in the schema.
type File struct {
	Claudelint *Claudelint  `hcl:"claudelint,block"`
	RulesKind  []RulesKind  `hcl:"rules,block"`
	Rules      []RuleBlock  `hcl:"rule,block"`
	Ignore     *IgnoreBlock `hcl:"ignore,block"`
	Output     *OutputBlock `hcl:"output,block"`
}

// Claudelint is the required version-declaring block. Its only v1
// field is "version"; the label-less form is deliberate so there is
// exactly one such block per file.
type Claudelint struct {
	Version string `hcl:"version"`
}

// RulesKind holds per-ArtifactKind defaults, e.g.
//
//	rules "skill" {
//	  default_severity = "warning"
//	}
//
// The label is the ArtifactKind string ("claude_md", "skill", ...).
type RulesKind struct {
	Kind            string    `hcl:"kind,label"`
	DefaultSeverity string    `hcl:"default_severity,optional"`
	Options         cty.Value `hcl:"options,optional"`
}

// RuleBlock overrides one specific rule:
//
//	rule "skills/body-size" {
//	  enabled  = false
//	  severity = "error"
//	  options  = { max_words = 1500 }
//	  paths    = ["legacy/**"]
//	}
//
// The label is the rule ID.
type RuleBlock struct {
	ID       string    `hcl:"id,label"`
	Enabled  *bool     `hcl:"enabled,optional"`
	Severity string    `hcl:"severity,optional"`
	Options  cty.Value `hcl:"options,optional"`
	Paths    []string  `hcl:"paths,optional"`
}

// IgnoreBlock is the global path-glob exclusion list:
//
//	ignore {
//	  paths = ["vendor/**", "node_modules/**"]
//	}
type IgnoreBlock struct {
	Paths []string `hcl:"paths,optional"`
}

// OutputBlock selects the reporter format:
//
//	output {
//	  format = "text"  # "json" or "github" also accepted
//	}
type OutputBlock struct {
	Format string `hcl:"format,optional"`
}
