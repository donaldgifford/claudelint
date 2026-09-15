package artifact_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// The table in docs/rules/rules.md is the only place a user learns why
// "removed in v2.0.64" says v2.0.64. A table that drifts from the Go
// map is worse than no table: it cites a version the diagnostic does
// not.

const rulesDocPath = "docs/rules/rules.md"

func TestDeprecatedToolsTableMatchesTheDoc(t *testing.T) {
	t.Parallel()

	rows := deprecatedToolRows(t)

	if len(rows) != len(artifact.DeprecatedTools) {
		t.Fatalf("%s lists %d tools, artifact.DeprecatedTools has %d",
			rulesDocPath, len(rows), len(artifact.DeprecatedTools))
	}

	for name, want := range artifact.DeprecatedTools {
		got, ok := rows[name]
		if !ok {
			t.Errorf("%s has no row for %q", rulesDocPath, name)

			continue
		}

		for _, f := range []struct {
			column    string
			got, want string
		}{
			{"status", got[0], string(want.Status)},
			{"version", got[1], want.Since},
			{"replacement", got[2], want.ReplacedBy},
			{"source", got[3], want.Source},
		} {
			if f.got != f.want {
				t.Errorf("%s: %s column = %q, want %q", name, f.column, f.got, f.want)
			}
		}
	}
}

// TestDeprecatedToolsAreConsistentWithKnownTools is the other half of
// the guardrail's relation, checked here so it holds even without the
// digest: a removed or renamed tool must be absent from the documented
// list, a deprecated one present, and every replacement must itself be
// a tool that exists.
func TestDeprecatedToolsAreConsistentWithKnownTools(t *testing.T) {
	t.Parallel()

	for name, d := range artifact.DeprecatedTools {
		_, known := artifact.KnownTools[name]

		switch d.Status {
		case artifact.ToolDeprecated:
			if !known {
				t.Errorf("%s is deprecated, so it is still documented and must stay in KnownTools", name)
			}
			if _, superseded := artifact.SupersededToolAdvice(name); superseded {
				t.Errorf("%s is deprecated but still documented; it must not produce a diagnostic", name)
			}
		case artifact.ToolRemoved, artifact.ToolRenamed:
			if _, superseded := artifact.SupersededToolAdvice(name); !superseded {
				t.Errorf("%s is %s but produces no diagnostic", name, d.Status)
			}
		default:
			t.Errorf("%s has an unrecognized status %q", name, d.Status)
		}

		if _, ok := artifact.KnownTools[d.ReplacedBy]; !ok {
			t.Errorf("%s points at %q, which is not a known tool", name, d.ReplacedBy)
		}
		if !strings.HasPrefix(d.Since, "v") {
			t.Errorf("%s: Since = %q, want a leading v", name, d.Since)
		}
		if strings.TrimSpace(d.Source) == "" {
			t.Errorf("%s: no Source, so the version it cites is unattributable", name)
		}
	}
}

// deprecatedToolRows reads the "Deprecated and removed tools" table out
// of the rules doc, keyed by tool name.
func deprecatedToolRows(t *testing.T) map[string][4]string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(rulesDocPath)))
	if err != nil {
		t.Fatalf("read the rules doc: %v", err)
	}

	const heading = "##### Deprecated and removed tools"

	body := string(raw)
	start := strings.Index(body, heading)
	if start < 0 {
		t.Fatalf("%s has no %q section", rulesDocPath, heading)
	}
	body = body[start+len(heading):]
	if end := strings.Index(body, "\n#"); end >= 0 {
		body = body[:end]
	}

	rows := make(map[string][4]string)
	for _, line := range strings.Split(body, "\n") {
		cells, ok := tableRow(line)
		if !ok || cells[0] == "Tool" || strings.HasPrefix(cells[0], "---") {
			continue
		}
		rows[cells[0]] = [4]string{cells[1], cells[2], cells[3], cells[4]}
	}

	if len(rows) == 0 {
		t.Fatalf("%s: the %q table has no rows", rulesDocPath, heading)
	}

	return rows
}

// tableRow splits a five-column Markdown row, stripping the code ticks
// the doc wraps names in.
func tableRow(line string) ([5]string, bool) {
	var out [5]string

	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return out, false
	}

	parts := strings.Split(strings.Trim(line, "|"), "|")
	if len(parts) != len(out) {
		return out, false
	}
	for i, p := range parts {
		out[i] = strings.Trim(strings.TrimSpace(p), "`")
	}

	return out, true
}
