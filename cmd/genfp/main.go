// Package main prints the current ruleset fingerprint. Used by the
// CI guardrail test and by release tooling to keep
// internal/rules/expected_fingerprint.txt aligned with the ruleset.
package main

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/rules"
	_ "github.com/donaldgifford/claudelint/internal/rules/all"
)

func main() {
	fmt.Print(rules.RulesetFingerprint())
}
