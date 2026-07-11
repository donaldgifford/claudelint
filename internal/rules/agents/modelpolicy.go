package agents

import (
	"fmt"
	"slices"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&modelPolicy{}) }

const (
	requireOption   = "require"
	allowlistOption = "allowlist"

	// modelInherit is the documented default model reference an
	// absent `model` key resolves to.
	modelInherit = "inherit"
)

// modelPolicy enforces a governance policy over agent `model`
// declarations. Opt-in via rules.OptIn (DESIGN-0005 §6): without a
// `rule "agents/model-policy"` block the engine never runs it.
//
// Exactly one option must be set once a block exists:
//
//	require = "inherit"     → every agent must inherit: compliant when
//	                          `model` is absent or `inherit`
//	allowlist = ["opus", …] → declared `model` must be in the list; an
//	                          absent key is evaluated as `inherit`
//
// Anything else — both options, neither, `require` not "inherit", or
// an allowlist entry that is not a valid model reference — emits a
// loud config-error diagnostic per artifact, mirroring
// mcp/server-allowlist: an explicit enable that cannot be evaluated is
// a misconfiguration, not a silent no-op.
type modelPolicy struct{}

func (*modelPolicy) ID() string                     { return "agents/model-policy" }
func (*modelPolicy) Category() string               { return categorySchema }
func (*modelPolicy) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*modelPolicy) OptIn() bool                    { return true }
func (*modelPolicy) DefaultOptions() map[string]any {
	// nil defaults declare the keys so config values overlay, while
	// letting Check distinguish "unset" from "set to empty".
	return map[string]any{requireOption: nil, allowlistOption: nil}
}

func (*modelPolicy) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindAgent}
}

func (*modelPolicy) HelpURI() string { return rules.DefaultHelpURI("agents/model-policy") }

func (r *modelPolicy) Check(ctx rules.Context, a artifact.Artifact) []diag.Diagnostic {
	ag, ok := a.(*artifact.Agent)
	if !ok {
		return nil
	}
	requireInherit, allowlist, confErr := parsePolicy(ctx)
	if confErr != "" {
		return []diag.Diagnostic{{
			RuleID:  r.ID(),
			Path:    ag.Path(),
			Range:   policyAnchor(ag),
			Message: confErr,
		}}
	}

	if requireInherit {
		if ag.Model == "" || ag.Model == modelInherit {
			return nil
		}
		return []diag.Diagnostic{{
			RuleID: r.ID(),
			Path:   ag.Path(),
			Range:  ag.Frontmatter.KeyRange("model"),
			Message: fmt.Sprintf(
				`model %q violates the require = "inherit" policy`, ag.Model),
		}}
	}

	effective := ag.Model
	if effective == "" {
		effective = modelInherit
	}
	if slices.Contains(allowlist, effective) {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID: r.ID(),
		Path:   ag.Path(),
		Range:  policyAnchor(ag),
		Message: fmt.Sprintf(
			"model %q is not in the configured model allowlist", effective),
	}}
}

// parsePolicy resolves the rule options into one of the two policy
// modes, or a config-error message when the block cannot be evaluated.
func parsePolicy(ctx rules.Context) (requireInherit bool, allowlist []string, confErr string) {
	rawReq := ctx.Option(requireOption)
	rawAllow := ctx.Option(allowlistOption)
	switch {
	case rawReq == nil && rawAllow == nil:
		return false, nil, `agents/model-policy enabled without options; set exactly one of ` +
			`require = "inherit" or allowlist = [...]`
	case rawReq != nil && rawAllow != nil:
		return false, nil,
			`agents/model-policy options "require" and "allowlist" are mutually exclusive; set exactly one`
	}
	if rawReq != nil {
		s, ok := rawReq.(string)
		if !ok || s != modelInherit {
			return false, nil, fmt.Sprintf(
				`agents/model-policy option "require" only supports "inherit", got %v`, rawReq)
		}
		return true, nil, ""
	}
	list, ok := asStringList(rawAllow)
	if !ok {
		return false, nil, `agents/model-policy option "allowlist" must be a list of strings`
	}
	for _, entry := range list {
		if !artifact.IsValidModelRef(entry) {
			return false, nil, fmt.Sprintf(
				"agents/model-policy allowlist entry %q is not a valid model reference", entry)
		}
	}
	return false, list, ""
}

// policyAnchor picks a stable non-zero range for diagnostics that are
// not tied to a declared `model` key: the model key when present, else
// the name key, else the body. File-level (0,0) ranges would break
// per-line suppression markers.
func policyAnchor(ag *artifact.Agent) diag.Range {
	if r := ag.Frontmatter.KeyRange("model"); !r.IsZero() {
		return r
	}
	if r := ag.Frontmatter.KeyRange("name"); !r.IsZero() {
		return r
	}
	return ag.Body
}

// asStringList coerces a rule-option value into a []string. HCL lists
// arrive as []any after cty conversion.
func asStringList(v any) ([]string, bool) {
	switch t := v.(type) {
	case []string:
		return t, true
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	}
	return nil, false
}
