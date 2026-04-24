// Package rules owns the ruleset's version metadata. RulesetVersion
// is a hand-bumped semver; RulesetFingerprint is derived from the
// currently-registered rules at call time so accidental drift fails
// the CI guardrail test added in phase 1.5.
package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// RulesetVersion is the hand-bumped semver of the built-in ruleset,
// independent of the binary version. It must be updated whenever a
// rule is added, removed, or changes its default severity or default
// options. The CI guardrail test TestRulesetFingerprint fails loudly
// if the ruleset content drifts without a corresponding bump.
const RulesetVersion = "v1.1.0"

// RulesetFingerprint returns a short hex hash of the registered rules'
// content. The hash covers:
//
//   - every rule's ID,
//   - its Category,
//   - its DefaultSeverity,
//   - its DefaultOptions keys/values,
//   - its AppliesTo kinds (sorted).
//
// Any change in any of those fields flips the fingerprint, so phase
// 1.5's CI guardrail test catches drift without requiring rule
// authors to touch a separate file.
func RulesetFingerprint() string {
	rules := All()
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID() < rules[j].ID() })

	var buf strings.Builder
	for _, r := range rules {
		fmt.Fprintf(&buf, "id=%s|cat=%s|sev=%d|", r.ID(), r.Category(), r.DefaultSeverity())

		kinds := make([]string, 0, len(r.AppliesTo()))
		for _, k := range r.AppliesTo() {
			kinds = append(kinds, string(k))
		}
		sort.Strings(kinds)
		fmt.Fprintf(&buf, "kinds=%s|", strings.Join(kinds, ","))

		opts := r.DefaultOptions()
		keys := make([]string, 0, len(opts))
		for k := range opts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&buf, "opt:%s=%v|", k, opts[k])
		}
		buf.WriteString("\n")
	}
	h := sha256.Sum256([]byte(buf.String()))
	return hex.EncodeToString(h[:])[:8]
}
