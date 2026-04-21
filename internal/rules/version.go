// Package rules holds the Rule interface, the registry, and the
// ruleset's version metadata. Phase 1.1 seeds only the version
// constants so cmd/claudelint version has a real home for its data;
// Rule, Context, Register, All, Get, and RulesetFingerprint land in
// phase 1.4.
package rules

// RulesetVersion is the hand-bumped semver of the built-in ruleset,
// independent of the binary version. It must be updated whenever a
// rule is added, removed, or changes its default severity or default
// options. The CI guardrail test added in phase 1.5 fails loudly if
// the ruleset content drifts without a corresponding bump.
const RulesetVersion = "v0.0.0"

// RulesetFingerprint is the short hash that pairs with RulesetVersion
// for end-user output, the same way go modules pair semver with a
// content hash. Phase 1.5 replaces this constant with a function that
// derives the value from the registered rules at init time; today it
// is the literal "unset" placeholder so downstream output shape is
// already stable.
const RulesetFingerprint = "unset"
