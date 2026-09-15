package upstream_test

import (
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

// The coverage table and the guardrail ask the same question from two
// directions. The guardrail compares the digest with the code and
// fails on a difference; the coverage table describes the same
// comparison for a reader. If they drift apart, the page starts saying
// something the test does not enforce.

func TestCoverageValidatesAgainstTheCommittedDigest(t *testing.T) {
	t.Parallel()

	digest, _, err := upstream.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error = %v", err)
	}

	if err := upstream.DefaultCoverage().Validate(digest); err != nil {
		t.Errorf("DefaultCoverage().Validate() error = %v", err)
	}
}

// TestCoverageMatchesTheGuardrail is the link coverage.go's doc comment
// promises: every digest path the guardrail compares has a coverage
// entry, so a section the test enforces is never rendered as one
// claudelint has no opinion about.
func TestCoverageMatchesTheGuardrail(t *testing.T) {
	t.Parallel()

	cov := upstream.DefaultCoverage()

	for _, path := range guardedPaths() {
		if _, ok := cov[path]; !ok {
			t.Errorf("the guardrail compares %q but the coverage table has no entry for it, "+
				"so the spec page renders it as \"no opinion\"", path)
		}
	}
}

// guardedPaths is every digest path TestGuardrail compares. It is
// written out rather than derived because a derived list would go stale
// in exactly the way this test exists to catch.
func guardedPaths() []string {
	return []string{
		"agents.enums.color",
		"agents.enums.effort",
		"agents.enums.isolation",
		"agents.enums.memory",
		"agents.enums.model",
		"agents.enums.permissionMode",
		"agents.frontmatter",
		"hooks.events",
		"hooks.handler_fields",
		"hooks.types",
		"marketplace.reserved_names",
		"marketplace.sources",
		"mcp.transports",
		"plugins.manifest_fields",
		"skills.frontmatter",
		"tools.builtin",
	}
}

func TestCoverageLookupIsThreeStates(t *testing.T) {
	t.Parallel()

	cov := upstream.DefaultCoverage()

	tests := []struct {
		name       string
		path, item string
		wantKnown  bool
		wantParsed bool
	}{
		{
			name: "a tool claudelint knows", path: "tools.builtin", item: "Read",
			wantKnown: true, wantParsed: true,
		},
		{
			name: "a documented field nothing reads", path: "skills.frontmatter", item: "paths",
			wantKnown: true, wantParsed: false,
		},
		{
			name: "a section with no coverage entry", path: "marketplace.fields", item: "name",
			wantKnown: false, wantParsed: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			known, parsed := cov.Lookup(tc.path, tc.item)
			if known != tc.wantKnown || parsed != tc.wantParsed {
				t.Errorf("Lookup(%q, %q) = (%t, %t), want (%t, %t)",
					tc.path, tc.item, known, parsed, tc.wantKnown, tc.wantParsed)
			}
		})
	}
}

// TestCoverageRejectsAnUnknownPath is the same guard Acknowledged has,
// for the same reason: a typo would render as "no opinion" forever.
func TestCoverageRejectsAnUnknownPath(t *testing.T) {
	t.Parallel()

	digest, _, err := upstream.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error = %v", err)
	}

	cov := upstream.Coverage{"hooks.imaginary": {"x": {}}}

	err = cov.Validate(digest)
	if err == nil {
		t.Fatal("Validate() accepted a path that is not in the digest")
	}
	if !strings.Contains(err.Error(), "hooks.imaginary") {
		t.Errorf("error does not name the bad path: %v", err)
	}
}
