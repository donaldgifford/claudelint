package reporter

import (
	"fmt"
	"io"
	"strings"

	"github.com/donaldgifford/claudelint/internal/diag"
)

// GitHub renders a Summary as GitHub Actions workflow commands so each
// diagnostic shows up as an annotation on the PR / job log:
//
//	::error file=path,line=2,col=1,title=skills/require-name::frontmatter missing "name"
//	::warning file=other.md,line=7,col=3,title=style/no-emoji::...
//	::notice file=x.md,line=1,col=1,title=info/something::...
//
// The workflow-command syntax requires literal `%`, `\r`, `\n`, `:`
// and `,` inside values to be percent-escaped; escapeValue handles
// that. A trailing summary line matches the text reporter so the
// annotation output still ends with a readable count.
func GitHub(w io.Writer, s Summary) error {
	for i := range s.Diagnostics {
		if err := writeGitHubLine(w, &s.Diagnostics[i]); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "%d diagnostics, %d files checked\n", len(s.Diagnostics), s.Files)
	return err
}

func writeGitHubLine(w io.Writer, d *diag.Diagnostic) error {
	cmd := severityToCommand(d.Severity)

	var params []string
	if d.Path != "" {
		params = append(params, "file="+escapeProperty(d.Path))
	}
	if !d.Range.Start.IsZero() {
		params = append(params, fmt.Sprintf("line=%d", d.Range.Start.Line))
		if d.Range.Start.Column > 0 {
			params = append(params, fmt.Sprintf("col=%d", d.Range.Start.Column))
		}
	}
	if d.RuleID != "" {
		params = append(params, "title="+escapeProperty(d.RuleID))
	}

	_, err := fmt.Fprintf(w, "::%s %s::%s\n", cmd, strings.Join(params, ","), escapeData(d.Message))
	return err
}

func severityToCommand(s diag.Severity) string {
	switch s {
	case diag.SeverityError:
		return "error"
	case diag.SeverityWarning:
		return "warning"
	default:
		return "notice"
	}
}

// escapeProperty percent-escapes runes that break the workflow-command
// property list. GitHub documents `%`, `\r`, `\n`, `:`, and `,` as
// required escapes inside property values.
func escapeProperty(v string) string {
	r := strings.NewReplacer(
		"%", "%25",
		"\r", "%0D",
		"\n", "%0A",
		":", "%3A",
		",", "%2C",
	)
	return r.Replace(v)
}

// escapeData percent-escapes runes that break the workflow-command
// data section (the part after `::`). Only `%`, `\r`, and `\n` need
// escaping; colons and commas are permitted inside messages.
func escapeData(v string) string {
	r := strings.NewReplacer(
		"%", "%25",
		"\r", "%0D",
		"\n", "%0A",
	)
	return r.Replace(v)
}
