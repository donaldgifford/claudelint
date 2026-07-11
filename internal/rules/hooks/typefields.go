package hooks

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&typeFields{}) }

// typeFields errors when a hook entry is missing the field its type
// requires: command hooks need `command`, http hooks need `url`,
// mcp_tool hooks need `server` and `tool`, prompt and agent hooks
// need `prompt`. A hook without its payload field is silently inert
// at runtime. Entries with an unknown declared type are left to
// hooks/type-known.
type typeFields struct{}

func (*typeFields) ID() string                     { return "hooks/type-fields" }
func (*typeFields) Category() string               { return categorySchema }
func (*typeFields) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*typeFields) DefaultOptions() map[string]any { return nil }
func (*typeFields) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindHook}
}

func (*typeFields) HelpURI() string { return rules.DefaultHelpURI("hooks/type-fields") }

func (r *typeFields) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	h, ok := a.(*artifact.Hook)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	for i := range h.Entries {
		e := &h.Entries[i]
		if e.Type != "" && !artifact.IsKnownHookType(e.Type) {
			continue
		}
		for _, missing := range missingTypeFields(e) {
			out = append(out, diag.Diagnostic{
				RuleID: r.ID(),
				Path:   h.Path(),
				Range:  entryAnchor(e),
				Message: fmt.Sprintf(
					"%s-type hook is missing required field %q",
					e.EffectiveType(), missing),
			})
		}
	}
	return out
}

// missingTypeFields lists the required-field names absent from e,
// keyed off the documented per-type payload fields.
func missingTypeFields(e *artifact.HookEntry) []string {
	var missing []string
	switch e.EffectiveType() {
	case typeCommand:
		if e.Command == "" {
			missing = append(missing, typeCommand)
		}
	case "http":
		if e.URL == "" {
			missing = append(missing, "url")
		}
	case "mcp_tool":
		if e.Server == "" {
			missing = append(missing, "server")
		}
		if e.Tool == "" {
			missing = append(missing, "tool")
		}
	case "prompt", "agent":
		if e.Prompt == "" {
			missing = append(missing, "prompt")
		}
	}
	return missing
}

// entryAnchor returns the first non-zero range the entry carries —
// the missing field itself has no range to point at, and file-level
// (0,0) ranges break per-line suppressions. The parser leaves
// EventRange unset, so the chain walks the per-key ranges instead.
func entryAnchor(e *artifact.HookEntry) diag.Range {
	for _, r := range []diag.Range{
		e.TypeRange, e.CommandRange, e.URLRange, e.TimeoutRange, e.MatcherRange,
	} {
		if !r.IsZero() {
			return r
		}
	}
	return e.EventRange
}
