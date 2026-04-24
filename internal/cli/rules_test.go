package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/rules"
)

// fixed subset of production rules used to assert shapes without
// coupling the test to every rule's exact behavior. These rules are
// part of the MVP and are very unlikely to be renamed without a
// conscious schema-version bump.
var rulesJSONFixedSubset = []string{
	"schema/parse",
	"schema/frontmatter-required",
	"mcp/no-unsafe-shell",
	"marketplace/plugins-nonempty",
}

func TestRulesTextOutputShape(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"rules"})

	if err := root.Execute(); err != nil {
		t.Fatalf("rules: %v", err)
	}

	got := buf.String()
	for _, header := range []string{"ID", "CATEGORY", "SEVERITY", "KINDS"} {
		if !strings.Contains(got, header) {
			t.Errorf("text output missing header %q: %s", header, got)
		}
	}
	for _, id := range rulesJSONFixedSubset {
		if !strings.Contains(got, id) {
			t.Errorf("text output missing rule %q", id)
		}
	}
}

func TestRulesJSONEnvelopeShape(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"rules", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("rules --json: %v", err)
	}

	var doc rulesDoc
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if doc.SchemaVersion != rulesJSONSchemaVersion {
		t.Errorf("schema_version = %q, want %q", doc.SchemaVersion, rulesJSONSchemaVersion)
	}
	if doc.RulesetVersion != rules.RulesetVersion {
		t.Errorf("ruleset_version = %q, want %q", doc.RulesetVersion, rules.RulesetVersion)
	}
	if doc.Fingerprint != rules.RulesetFingerprint() {
		t.Errorf("fingerprint = %q, want %q", doc.Fingerprint, rules.RulesetFingerprint())
	}
	if len(doc.Rules) == 0 {
		t.Fatalf("rules array is empty")
	}

	byID := make(map[string]ruleDoc, len(doc.Rules))
	for _, r := range doc.Rules {
		byID[r.ID] = r
	}
	for _, want := range rulesJSONFixedSubset {
		got, ok := byID[want]
		if !ok {
			t.Errorf("rules array missing %q", want)
			continue
		}
		if got.Category == "" {
			t.Errorf("%s: category empty", want)
		}
		if got.DefaultSeverity == "" {
			t.Errorf("%s: default_severity empty", want)
		}
		if got.HelpURI == "" {
			t.Errorf("%s: help_uri empty", want)
		}
		if got.DefaultOptions == nil {
			t.Errorf("%s: default_options is nil, want empty object", want)
		}
		if got.AppliesTo == nil {
			t.Errorf("%s: applies_to is nil, want array", want)
		}
	}
}

func TestRulesJSONSingleRuleEnvelope(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"rules", "mcp/no-unsafe-shell", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("rules mcp/no-unsafe-shell --json: %v", err)
	}

	var doc rulesDoc
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if len(doc.Rules) != 1 {
		t.Fatalf("rules length = %d, want 1", len(doc.Rules))
	}
	r := doc.Rules[0]
	if r.ID != "mcp/no-unsafe-shell" {
		t.Errorf("id = %q, want mcp/no-unsafe-shell", r.ID)
	}
	if r.Category != "security" {
		t.Errorf("category = %q, want security", r.Category)
	}
	if r.DefaultSeverity != "error" {
		t.Errorf("default_severity = %q, want error", r.DefaultSeverity)
	}
	if len(r.AppliesTo) != 1 || r.AppliesTo[0] != "mcp_server" {
		t.Errorf("applies_to = %v, want [mcp_server]", r.AppliesTo)
	}
	if !strings.HasPrefix(r.HelpURI, "https://") {
		t.Errorf("help_uri = %q, want an https URL", r.HelpURI)
	}
}

func TestRulesJSONUnknownRuleErrors(t *testing.T) {
	root := newRootCmd(BuildInfo{Version: "v0", Commit: "c"})
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"rules", "does/not-exist", "--json"})

	err := root.Execute()
	if err == nil {
		t.Fatalf("unknown rule should error in --json mode")
	}
	if !strings.Contains(err.Error(), "unknown rule") {
		t.Errorf("error = %v, want unknown rule", err)
	}
}
