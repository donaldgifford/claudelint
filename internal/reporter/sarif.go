package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// SARIFVersion is the SARIF specification version this reporter
// produces. The vendored schema under testdata/ matches this value.
const SARIFVersion = "2.1.0"

// sarifSchemaURI is the JSON Schema URI emitted as `$schema`. Keep this
// in sync with the vendored testdata/sarif-2.1.0.json.
const sarifSchemaURI = "https://json.schemastore.org/sarif-2.1.0.json"

// sarifInformationURI is the tool's homepage, emitted on
// `runs[0].tool.driver.informationUri`. Matches the help URI base.
const sarifInformationURI = "https://github.com/donaldgifford/claudelint"

// SARIFOptions controls the SARIF reporter. ToolVersion is the
// claudelint binary version (typically injected via -ldflags); it
// lands in `runs[0].tool.driver.version`. An empty ToolVersion falls
// back to the ruleset version so the document is still valid.
type SARIFOptions struct {
	ToolVersion string
}

// SARIF renders a Summary as a SARIF 2.1.0 log to w. The structure
// mirrors JSON(): one run, one driver, rules pre-populated from the
// registry so every result's ruleId resolves. Output ends with a
// trailing newline for parity with other reporters.
func SARIF(w io.Writer, s Summary, opts SARIFOptions) error {
	toolVersion := opts.ToolVersion
	if toolVersion == "" {
		toolVersion = rules.RulesetVersion
	}

	doc := sarifLog{
		Schema:  sarifSchemaURI,
		Version: SARIFVersion,
		Runs: []sarifRun{{
			Tool: sarifTool{
				Driver: sarifDriver{
					Name:           "claudelint",
					Version:        toolVersion,
					InformationURI: sarifInformationURI,
					Rules:          buildSARIFRules(),
				},
			},
			Results: buildSARIFResults(s.Diagnostics),
		}},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("encode sarif report: %w", err)
	}
	return nil
}

// buildSARIFRules materializes every registered rule as a
// reportingDescriptor. Including every rule (not just those that
// produced diagnostics) lets SARIF consumers render a full rule
// catalog — matches what golangci-lint and semgrep emit.
func buildSARIFRules() []sarifReportingDescriptor {
	all := rules.All()
	sort.Slice(all, func(i, j int) bool { return all[i].ID() < all[j].ID() })
	out := make([]sarifReportingDescriptor, 0, len(all))
	for _, r := range all {
		out = append(out, sarifReportingDescriptor{
			ID:   r.ID(),
			Name: sarifRuleName(r.ID()),
			ShortDescription: sarifMessage{
				Text: r.ID(),
			},
			HelpURI: r.HelpURI(),
			DefaultConfiguration: &sarifConfiguration{
				Level: severityToSARIF(r.DefaultSeverity()),
			},
		})
	}
	return out
}

// buildSARIFResults maps each diagnostic into a SARIF result object.
// Range.IsZero() → file-level result (no region), matching how the
// other reporters handle parse errors.
func buildSARIFResults(ds []diag.Diagnostic) []sarifResult {
	if len(ds) == 0 {
		return []sarifResult{}
	}
	out := make([]sarifResult, 0, len(ds))
	for i := range ds {
		d := &ds[i]
		out = append(out, sarifResult{
			RuleID:    d.RuleID,
			Level:     severityToSARIF(d.Severity),
			Message:   sarifMessage{Text: d.Message},
			Locations: []sarifLocation{buildSARIFLocation(d)},
		})
	}
	return out
}

func buildSARIFLocation(d *diag.Diagnostic) sarifLocation {
	loc := sarifLocation{
		PhysicalLocation: sarifPhysicalLocation{
			ArtifactLocation: sarifArtifactLocation{
				URI: d.Path,
			},
		},
	}
	if d.Range.Start.IsZero() {
		return loc
	}
	region := &sarifRegion{
		StartLine:   d.Range.Start.Line,
		StartColumn: d.Range.Start.Column,
	}
	if !d.Range.End.IsZero() {
		region.EndLine = d.Range.End.Line
		region.EndColumn = d.Range.End.Column
	}
	loc.PhysicalLocation.Region = region
	return loc
}

// SARIF `level` literal vocabulary. Defined once so goconst stays
// happy and downstream readers can grep for the exact strings.
const (
	sarifLevelError   = "error"
	sarifLevelWarning = "warning"
	sarifLevelNote    = "note"
)

// severityToSARIF maps claudelint severity to SARIF's `level` vocab.
// SARIF's "note" is the nearest equivalent to our Info.
func severityToSARIF(s diag.Severity) string {
	switch s {
	case diag.SeverityError:
		return sarifLevelError
	case diag.SeverityWarning:
		return sarifLevelWarning
	default:
		return sarifLevelNote
	}
}

// sarifRuleName returns the human-readable `name` field for a rule's
// reportingDescriptor. Rule IDs are category-prefixed (`mcp/no-unsafe-shell`);
// SARIF's `name` is conventionally a short, unprefixed identifier.
func sarifRuleName(id string) string {
	if _, name, ok := strings.Cut(id, "/"); ok {
		return name
	}
	return id
}

// --- SARIF document shape ---
// Structs mirror the subset of the SARIF 2.1.0 spec the reporter
// emits. Field order and json tags are part of the stability contract;
// the vendored schema under testdata/ validates it.

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string                     `json:"name"`
	Version        string                     `json:"version"`
	InformationURI string                     `json:"informationUri"`
	Rules          []sarifReportingDescriptor `json:"rules"`
}

type sarifReportingDescriptor struct {
	ID                   string              `json:"id"`
	Name                 string              `json:"name"`
	ShortDescription     sarifMessage        `json:"shortDescription"`
	HelpURI              string              `json:"helpUri"`
	DefaultConfiguration *sarifConfiguration `json:"defaultConfiguration,omitempty"`
}

type sarifConfiguration struct {
	Level string `json:"level"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           *sarifRegion          `json:"region,omitempty"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine,omitempty"`
	EndColumn   int `json:"endColumn,omitempty"`
}
