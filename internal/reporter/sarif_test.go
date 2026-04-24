package reporter

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// sarifSchemaPath is the vendored SARIF 2.1.0 JSON Schema. Keeping it
// in testdata/ means `make ci` validates SARIF output without hitting
// the network.
const sarifSchemaPath = "testdata/sarif-2.1.0.json"

// loadSARIFSchema compiles the vendored schema once per test invocation.
// Individual tests call through Validate to keep error paths readable.
func loadSARIFSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	raw, err := os.ReadFile(sarifSchemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(sarifSchemaPath, bytes.NewReader(raw)); err != nil {
		t.Fatalf("add resource: %v", err)
	}
	sch, err := c.Compile(sarifSchemaPath)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

// validateSARIF runs schema validation on the reporter output. Any
// validation failure is a test failure — every SARIF doc we emit must
// conform to 2.1.0.
func validateSARIF(t *testing.T, sch *jsonschema.Schema, got []byte) {
	t.Helper()
	var payload any
	if err := json.Unmarshal(got, &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, got)
	}
	if err := sch.Validate(payload); err != nil {
		t.Fatalf("schema validation failed: %v\n%s", err, got)
	}
}

func TestSARIFEmptySummaryValidates(t *testing.T) {
	sch := loadSARIFSchema(t)
	var buf bytes.Buffer
	if err := SARIF(&buf, Summary{}, SARIFOptions{ToolVersion: "v0.2.0-test"}); err != nil {
		t.Fatalf("SARIF: %v", err)
	}
	validateSARIF(t, sch, buf.Bytes())
}

func TestSARIFHasSchemaAndVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := SARIF(&buf, Summary{}, SARIFOptions{ToolVersion: "v0.2.0-test"}); err != nil {
		t.Fatalf("SARIF: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got, want := doc["$schema"], sarifSchemaURI; got != want {
		t.Errorf("$schema = %v, want %v", got, want)
	}
	if got, want := doc["version"], SARIFVersion; got != want {
		t.Errorf("version = %v, want %v", got, want)
	}
}

func TestSARIFPopulatesToolDriverRules(t *testing.T) {
	var buf bytes.Buffer
	if err := SARIF(&buf, Summary{}, SARIFOptions{ToolVersion: "v9.9.9"}); err != nil {
		t.Fatalf("SARIF: %v", err)
	}
	var doc struct {
		Runs []struct {
			Tool struct {
				Driver struct {
					Name    string `json:"name"`
					Version string `json:"version"`
					Rules   []struct {
						ID      string `json:"id"`
						Name    string `json:"name"`
						HelpURI string `json:"helpUri"`
					} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(doc.Runs) != 1 {
		t.Fatalf("runs = %d, want 1", len(doc.Runs))
	}
	driver := doc.Runs[0].Tool.Driver
	if driver.Name != "claudelint" {
		t.Errorf("driver.name = %q, want claudelint", driver.Name)
	}
	if driver.Version != "v9.9.9" {
		t.Errorf("driver.version = %q, want v9.9.9", driver.Version)
	}
	if len(driver.Rules) != len(rules.All()) {
		t.Errorf("driver.rules length = %d, want %d (every registered rule)",
			len(driver.Rules), len(rules.All()))
	}
	for _, r := range driver.Rules {
		if r.ID == "" || r.Name == "" || r.HelpURI == "" {
			t.Errorf("rule %q has empty id/name/helpUri: %+v", r.ID, r)
		}
	}
}

func TestSARIFResultsMapEveryDiagnostic(t *testing.T) {
	sch := loadSARIFSchema(t)
	ds := []diag.Diagnostic{
		{
			RuleID:   "mcp/no-unsafe-shell",
			Severity: diag.SeverityError,
			Path:     ".mcp.json",
			Range:    diag.Range{Start: diag.Position{Line: 4, Column: 18}},
			Message:  `MCP server "gh" invokes "bash -c"`,
		},
		{
			RuleID:   "marketplace/author-required",
			Severity: diag.SeverityInfo,
			Path:     ".claude-plugin/marketplace.json",
			Message:  "author is missing",
		},
	}
	var buf bytes.Buffer
	if err := SARIF(&buf, Summary{Diagnostics: ds, Files: 2}, SARIFOptions{ToolVersion: "v0"}); err != nil {
		t.Fatalf("SARIF: %v", err)
	}
	validateSARIF(t, sch, buf.Bytes())

	var doc struct {
		Runs []struct {
			Results []struct {
				RuleID    string `json:"ruleId"`
				Level     string `json:"level"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region *struct {
							StartLine   int `json:"startLine"`
							StartColumn int `json:"startColumn"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	results := doc.Runs[0].Results
	if len(results) != len(ds) {
		t.Fatalf("results = %d, want %d", len(results), len(ds))
	}

	// First result: has a region (range was set).
	if results[0].RuleID != "mcp/no-unsafe-shell" || results[0].Level != "error" {
		t.Errorf("first result = %+v", results[0])
	}
	if r := results[0].Locations[0].PhysicalLocation.Region; r == nil {
		t.Errorf("first result should carry a region")
	} else if r.StartLine != 4 || r.StartColumn != 18 {
		t.Errorf("region = %+v, want line=4 col=18", r)
	}

	// Second result: file-level, no region.
	if results[1].RuleID != "marketplace/author-required" || results[1].Level != "note" {
		t.Errorf("second result = %+v", results[1])
	}
	if results[1].Locations[0].PhysicalLocation.Region != nil {
		t.Errorf("file-level result should have no region")
	}
}

func TestSARIFRuleNameStripsCategoryPrefix(t *testing.T) {
	cases := []struct {
		id, want string
	}{
		{"mcp/no-unsafe-shell", "no-unsafe-shell"},
		{"schema/parse", "parse"},
		{"unprefixed", "unprefixed"},
	}
	for _, tc := range cases {
		if got := sarifRuleName(tc.id); got != tc.want {
			t.Errorf("sarifRuleName(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

func TestSARIFFallsBackToRulesetVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := SARIF(&buf, Summary{}, SARIFOptions{}); err != nil {
		t.Fatalf("SARIF: %v", err)
	}
	if !strings.Contains(buf.String(), `"version": "`+rules.RulesetVersion+`"`) {
		t.Errorf("empty ToolVersion should fall back to ruleset version %q: %s",
			rules.RulesetVersion, buf.String())
	}
}
