package artifact

import "fmt"

// A tool that leaves the reference table is not the same as a tool that
// never existed, and the difference is what a user needs to hear.
//
// "Unknown tool: BashOutput" invites the reader to check their spelling.
// "BashOutput was removed in v2.0.64; use TaskOutput" tells them what to
// write instead. The tools-known rules consult this table before
// KnownTools so the second message is the one they get.
//
// The Claude Code changelog records some of these and not others. Where
// there is no entry, Source records the docs version marker at which the
// tool was last seen in the reference table, and says so, rather than
// inventing a release.

// toolTaskOutput is named because three entries point at it: the two
// removals it replaced and its own deprecation.
const toolTaskOutput = "TaskOutput"

// ToolStatus says what became of a tool.
type ToolStatus string

// Tool statuses.
const (
	// ToolRenamed means the tool still exists under another name.
	ToolRenamed ToolStatus = "renamed"
	// ToolRemoved means the tool is gone; ReplacedBy names the closest
	// current equivalent.
	ToolRemoved ToolStatus = "removed"
	// ToolDeprecated means the tool is still documented and still works,
	// but something else is now preferred.
	ToolDeprecated ToolStatus = "deprecated"
)

// DeprecatedTool records what happened to one tool.
type DeprecatedTool struct {
	// Status is renamed, removed, or deprecated.
	Status ToolStatus
	// Since is the version at which it happened, with a leading "v".
	Since string
	// ReplacedBy is the tool to use instead.
	ReplacedBy string
	// Source says where Since came from: a changelog entry, or the docs
	// marker at which the tool left the reference table.
	Source string
}

// Advice renders the one-line explanation a rule puts in its message.
func (d DeprecatedTool) Advice(name string) string {
	switch d.Status {
	case ToolRenamed:
		return fmt.Sprintf("%s was renamed to %s in %s", name, d.ReplacedBy, d.Since)
	case ToolRemoved:
		return fmt.Sprintf("%s was removed in %s; use %s", name, d.Since, d.ReplacedBy)
	case ToolDeprecated:
		return fmt.Sprintf("%s is deprecated as of %s; prefer %s", name, d.Since, d.ReplacedBy)
	default:
		return name + " is no longer current"
	}
}

// DeprecatedTools maps a tool name to its fate. The upstream guardrail
// checks each entry against the documented tool list: a removed or
// renamed tool must be absent from it, and a deprecated one present.
var DeprecatedTools = map[string]DeprecatedTool{
	"BashOutput": {
		Status:     ToolRemoved,
		Since:      "v2.0.64",
		ReplacedBy: toolTaskOutput,
		Source:     `Claude Code changelog, "Unshipped BashOutputTool"`,
	},
	"KillShell": {
		Status:     ToolRemoved,
		Since:      "v2.1.268",
		ReplacedBy: "TaskStop",
		Source:     "no changelog entry; last docs marker carrying it in the tools reference",
	},
	"MultiEdit": {
		Status:     ToolRemoved,
		Since:      "v2.1.268",
		ReplacedBy: "Edit",
		Source:     "no changelog entry; last docs marker carrying it in the tools reference",
	},
	"Task": {
		Status:     ToolRenamed,
		Since:      "v2.1.63",
		ReplacedBy: "Agent",
		Source:     "INV-0006; the runtime still accepts the old name",
	},
	toolTaskOutput: {
		Status:     ToolDeprecated,
		Since:      "v2.1.83",
		ReplacedBy: "Read",
		Source:     "Claude Code changelog",
	},
}

// LookupDeprecatedTool returns the record for a tool that is no longer
// current. ok is false for a tool with nothing to say about it, which
// is almost all of them.
func LookupDeprecatedTool(name string) (DeprecatedTool, bool) {
	d, ok := DeprecatedTools[name]

	return d, ok
}

// IsSupersededTool reports whether a tool has been renamed or removed,
// as opposed to merely deprecated. A superseded tool should not be
// declared in new artifacts; a deprecated one still works.
func IsSupersededTool(name string) bool {
	d, ok := DeprecatedTools[name]

	return ok && (d.Status == ToolRenamed || d.Status == ToolRemoved)
}
