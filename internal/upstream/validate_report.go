package upstream

import (
	"fmt"
	"strings"
)

// The report has three renderings for three readers: a terminal, the
// issue script, and whatever consumes the JSON artifact. They carry the
// same facts, and the Markdown one leads with the disagreements because
// that is the only part anyone acts on.

// HasDisagreement reports whether any fixture and the runtime disagree.
func (r *RuntimeReport) HasDisagreement() bool {
	return len(r.Disagreements()) > 0
}

// Disagreements returns only the rows worth acting on.
func (r *RuntimeReport) Disagreements() []RuntimeResult {
	var out []RuntimeResult
	for i := range r.Results {
		if !r.Results[i].Agrees {
			out = append(out, r.Results[i])
		}
	}

	return out
}

// Summary is the one line printed when the report went to a file.
func (r *RuntimeReport) Summary() string {
	bad := len(r.Disagreements())
	if bad == 0 {
		return fmt.Sprintf("%s agrees on all %d fixtures", r.ClaudeVersion, len(r.Results))
	}

	return fmt.Sprintf("%s disagrees on %d of %d fixtures", r.ClaudeVersion, bad, len(r.Results))
}

// Text is the terse rendering for a terminal.
func (r *RuntimeReport) Text() string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s\n", r.Summary())
	for i := range r.Results {
		res := &r.Results[i]
		mark := "ok"
		if !res.Agrees {
			mark = "DISAGREES"
		}
		fmt.Fprintf(&b, "  %-9s %s (expect %s, runtime %s)\n",
			mark, res.Path, res.Expect, passFail(res.Success))
		writeTextMessages(&b, res)
	}

	return b.String()
}

// writeTextMessages quotes the runtime's own words under a row that
// disagrees. A row that agrees needs no explanation.
func writeTextMessages(b *strings.Builder, res *RuntimeResult) {
	if res.Agrees {
		return
	}
	for _, msg := range res.Errors {
		fmt.Fprintf(b, "    error:   %s\n", msg)
	}
	for _, msg := range res.Warnings {
		fmt.Fprintf(b, "    warning: %s\n", msg)
	}
	if res.Probe == ProbeContentsEmpty && !res.ProbeOK {
		fmt.Fprintf(b, "    probe:   the validator walked %d contents entries; "+
			"claudelint's coverage claim assumed none\n", res.Contents)
	}
}

// JSON is the machine rendering, canonically encoded.
func (r *RuntimeReport) JSON() ([]byte, error) {
	var b strings.Builder
	if err := encodeJSON(&b, r); err != nil {
		return nil, err
	}

	return []byte(b.String()), nil
}

// Markdown is what the issue script pastes into a comment.
func (r *RuntimeReport) Markdown() string {
	var b strings.Builder

	b.WriteString("## Runtime validator\n\n")
	fmt.Fprintf(&b, "Validated %d fixtures against `%s`.\n\n", len(r.Results), r.ClaudeVersion)

	bad := r.Disagreements()
	if len(bad) == 0 {
		b.WriteString("No disagreements: the runtime accepted every fixture claudelint calls valid " +
			"and rejected every one it calls invalid.\n")

		return b.String()
	}

	fmt.Fprintf(&b, "**%d disagreement(s).** Each one is either a claudelint fixture that is wrong "+
		"or a runtime change worth following.\n\n", len(bad))

	for i := range bad {
		writeMarkdownResult(&b, &bad[i])
	}

	return b.String()
}

// writeMarkdownResult renders one disagreeing fixture.
func writeMarkdownResult(b *strings.Builder, res *RuntimeResult) {
	fmt.Fprintf(b, "### `%s`\n\n", res.Path)
	fmt.Fprintf(b, "Kind `%s`; expected the runtime to %s, it %sed.\n",
		res.Kind, res.Expect, passFail(res.Success))
	if res.Note != "" {
		fmt.Fprintf(b, "\nFixture note: %s\n", res.Note)
	}

	writeMarkdownMessages(b, "Errors", res.Errors)
	writeMarkdownMessages(b, "Warnings", res.Warnings)

	if res.Probe == ProbeContentsEmpty && !res.ProbeOK {
		fmt.Fprintf(b, "\nThe validator walked %d contents entries. claudelint's coverage "+
			"claim assumed it walks none, so that claim needs revisiting.\n", res.Contents)
	}

	b.WriteString("\n")
}

// writeMarkdownMessages quotes the runtime verbatim, because the person
// reading this in CI cannot re-run the binary.
func writeMarkdownMessages(b *strings.Builder, title string, msgs []string) {
	if len(msgs) == 0 {
		return
	}

	fmt.Fprintf(b, "\n%s:\n\n", title)
	for _, msg := range msgs {
		fmt.Fprintf(b, "- %s\n", msg)
	}
}

// passFail renders a boolean the way the report talks about it.
func passFail(ok bool) string {
	if ok {
		return ExpectPass
	}

	return ExpectFail
}
