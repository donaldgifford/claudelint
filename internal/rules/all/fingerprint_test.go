package all_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/rules"
	_ "github.com/donaldgifford/claudelint/internal/rules/all"
)

// TestRulesetFingerprint is the CI guardrail. It re-computes the
// fingerprint from the currently-registered rules and compares it
// against the value checked into internal/rules/expected_fingerprint.txt.
// Any drift fails loudly with a message telling the developer exactly
// what to do: bump RulesetVersion and update the expected file.
//
// This test lives under internal/rules/all rather than internal/rules
// so it runs in a test binary that does *not* share state with
// internal/rules/rules_test.go (which calls Reset() to isolate stub
// registrations). Keeping it here guarantees the full ruleset is
// registered for this assertion.
func TestRulesetFingerprint(t *testing.T) {
	want, err := readExpectedFingerprint()
	if err != nil {
		t.Fatalf("read expected fingerprint: %v", err)
	}
	got := rules.RulesetFingerprint()
	if got != want {
		t.Fatalf(
			"ruleset fingerprint drift: got %q, want %q\n\n"+
				"The registered rule set has changed (ID, category, severity, "+
				"options, or AppliesTo). Bump rules.RulesetVersion and update "+
				"internal/rules/expected_fingerprint.txt with the new value:\n\n"+
				"    %s\n",
			got, want, got,
		)
	}
}

func readExpectedFingerprint() (string, error) {
	// internal/rules/all → ../expected_fingerprint.txt lives in
	// internal/rules. Using filepath.Join with the upward traversal
	// keeps the path explicit for readers.
	p := filepath.Join("..", "expected_fingerprint.txt")
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
