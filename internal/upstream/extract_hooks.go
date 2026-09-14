package upstream

import (
	"fmt"
	"regexp"
	"strings"
)

// Anchors on the hooks page.
var (
	anchorHookEvents       = headingAnchor(2, "Hook events")
	anchorHookHandler      = headingAnchor(3, "Hook handler fields")
	anchorHookCommonFields = headingAnchor(4, "Common fields")
)

// handlerSections maps each handler-field subsection to the digest keys
// it fills. The prompt and agent handlers share one table.
var handlerSections = []struct {
	anchor *regexp.Regexp
	keys   []string
}{
	{anchor: anchorHookCommonFields, keys: []string{"common"}},
	{anchor: headingAnchor(4, "Command hook fields"), keys: []string{"command"}},
	{anchor: headingAnchor(4, "HTTP hook fields"), keys: []string{"http"}},
	{anchor: headingAnchor(4, "MCP tool hook fields"), keys: []string{"mcp_tool"}},
	{anchor: headingAnchor(4, "Prompt and agent hook fields"), keys: []string{"prompt", "agent"}},
}

// hookEventHeadingLevel is the heading level at which each hook event
// is documented under the "Hook events" section.
const hookEventHeadingLevel = 3

// timeoutDefault matches "600 for `command`, `http`, and `mcp_tool`",
// the shape the timeout row uses for its per-type defaults.
var timeoutDefault = regexp.MustCompile(`(\d+)\s+for\s+`)

// timeoutOverride matches "to 30 on [`UserPromptSubmit`](...)", the
// shape used for per-event overrides.
var timeoutOverride = regexp.MustCompile(`to\s+(\d+)\s+on\s+`)

// hooksEvents reads the documented hook event names, one per heading
// under the hook events section.
type hooksEvents struct{}

var _ Extractor = hooksEvents{}

func (hooksEvents) Section() string        { return "hooks.events" }
func (hooksEvents) Source() string         { return SrcDocsHooks }
func (hooksEvents) Anchor() *regexp.Regexp { return anchorHookEvents }

func (hooksEvents) Extract(section string, out *Digest) error {
	out.Hooks.Events = Headings(section, hookEventHeadingLevel)

	return nonEmpty(out.Hooks.Events, "hook events")
}

// hooksTypes reads the handler types from the type row of the common
// fields table.
type hooksTypes struct{}

var _ Extractor = hooksTypes{}

func (hooksTypes) Section() string        { return "hooks.types" }
func (hooksTypes) Source() string         { return SrcDocsHooks }
func (hooksTypes) Anchor() *regexp.Regexp { return anchorHookHandler }

func (hooksTypes) Extract(section string, out *Digest) error {
	desc, err := commonFieldDescription(section, colType)
	if err != nil {
		return err
	}

	types := BacktickedTokens(desc)
	for i, tok := range types {
		types[i] = unquote(tok)
	}

	out.Hooks.Types = types

	return nonEmpty(types, "hook handler types")
}

// hooksHandlerFields reads the field table of each handler type.
type hooksHandlerFields struct{}

var _ Extractor = hooksHandlerFields{}

func (hooksHandlerFields) Section() string        { return "hooks.handler_fields" }
func (hooksHandlerFields) Source() string         { return SrcDocsHooks }
func (hooksHandlerFields) Anchor() *regexp.Regexp { return anchorHookHandler }

func (hooksHandlerFields) Extract(section string, out *Digest) error {
	byType := make(map[string][]Field, len(handlerSections))

	for _, hs := range handlerSections {
		sub, err := subSection(section, hs.anchor)
		if err != nil {
			return err
		}

		t, err := firstTableWithHeaders(sub, colField, colRequired)
		if err != nil {
			return err
		}

		rows := fieldRows(t, "", "")
		if err := nonEmpty(rows, hs.anchor.String()); err != nil {
			return err
		}

		for _, key := range hs.keys {
			byType[key] = rows
		}
	}

	out.Hooks.HandlerFields = byType

	return nonEmptyMap(byType, "hook handler fields")
}

// hooksTimeoutDefaults reads the documented timeouts out of the prose
// in the timeout row: a per-type default list, then the events whose
// default is lowered.
type hooksTimeoutDefaults struct{}

var _ Extractor = hooksTimeoutDefaults{}

func (hooksTimeoutDefaults) Section() string        { return "hooks.timeout_defaults" }
func (hooksTimeoutDefaults) Source() string         { return SrcDocsHooks }
func (hooksTimeoutDefaults) Anchor() *regexp.Regexp { return anchorHookHandler }

func (hooksTimeoutDefaults) Extract(section string, out *Digest) error {
	desc, err := commonFieldDescription(section, "timeout")
	if err != nil {
		return err
	}

	out.Hooks.TimeoutDefaults = TimeoutDefaults{
		ByType:  timeoutsFrom(desc, timeoutDefault),
		ByEvent: timeoutsFrom(desc, timeoutOverride),
	}

	if len(out.Hooks.TimeoutDefaults.ByType) == 0 {
		return fmt.Errorf("%w: hook timeout defaults by type", ErrEmptySection)
	}

	return nil
}

// timeoutsFrom finds each "<N> for" or "to <N> on" clause and assigns
// that number to every code span between it and the next clause.
func timeoutsFrom(desc string, clause *regexp.Regexp) map[string]int {
	matches := clause.FindAllStringSubmatchIndex(desc, -1)
	if len(matches) == 0 {
		return nil
	}

	out := make(map[string]int)

	for i, m := range matches {
		end := len(desc)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}

		seconds := atoi([]byte(desc[m[2]:m[3]]))
		for _, name := range BacktickedTokens(clauseScope(desc[m[1]:end])) {
			out[unquote(name)] = seconds
		}
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// clauseScope trims a clause at the first sentence boundary, so prose
// that follows the list does not contribute names.
func clauseScope(s string) string {
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i]
	}

	return s
}

// commonFieldDescription returns the description cell of one row of the
// common hook fields table.
func commonFieldDescription(section, field string) (string, error) {
	sub, err := subSection(section, anchorHookCommonFields)
	if err != nil {
		return "", err
	}

	t, err := firstTableWithHeaders(sub, colField, colRequired, "description")
	if err != nil {
		return "", err
	}

	nameCol := t.Column(colField)
	descCol := t.Column("description")

	for i := range t.Rows {
		if Name(t.Cell(i, nameCol)) == field {
			return t.Cell(i, descCol), nil
		}
	}

	return "", fmt.Errorf("%w: no %q row in the common hook fields table", ErrEmptySection, field)
}
