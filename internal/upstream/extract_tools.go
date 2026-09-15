package upstream

import (
	"regexp"
	"strings"
)

// anchorToolsPage is the tools reference page title. The built-in tool
// table is the first one after it, before any section heading.
var anchorToolsPage = mustAnchor(`(?m)^# Tools reference\s*$`)

// colTool and colToolDescription are the columns of the tool table.
const (
	colTool            = "tool"
	colToolDescription = "description"
)

// deprecatedPrefix is the documentation's own marker for a tool that is
// still listed but on its way out. It is read as a prefix rather than a
// substring so a description that merely mentions deprecating something
// else does not count.
const deprecatedPrefix = "deprecated"

// toolsBuiltin reads the built-in tool reference table.
//
// It fills two digest sections: every tool name, and the subset the
// documentation marks deprecated. The deprecated subset is what the
// Phase 2 guardrail checks artifact.DeprecatedTools against, so a
// removed tool that is still in that table shows up as a stale row.
type toolsBuiltin struct{}

var _ Extractor = toolsBuiltin{}

func (toolsBuiltin) Section() string        { return "tools.builtin" }
func (toolsBuiltin) Source() string         { return SrcDocsTools }
func (toolsBuiltin) Anchor() *regexp.Regexp { return anchorToolsPage }

func (toolsBuiltin) Extract(section string, out *Digest) error {
	t, err := firstTableWithHeaders(section, colTool, colToolDescription)
	if err != nil {
		return err
	}

	nameCol := t.Column(colTool)
	descCol := t.Column(colToolDescription)

	names := make([]string, 0, len(t.Rows))

	var deprecated []string

	for i := range t.Rows {
		name := Name(t.Cell(i, nameCol))
		if name == "" {
			continue
		}
		names = append(names, name)

		desc := strings.ToLower(strings.TrimSpace(Plain(t.Cell(i, descCol))))
		if strings.HasPrefix(desc, deprecatedPrefix) {
			deprecated = append(deprecated, name)
		}
	}

	out.Tools.Builtin = names
	out.Tools.Deprecated = deprecated

	return nonEmpty(names, "built-in tools")
}
