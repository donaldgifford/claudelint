package mcp

import (
	"slices"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&serverAllowlist{}) }

const allowlistOption = "allowlist"

// serverAllowlist errors when an MCP server's name is not in the
// user's configured allowlist. The rule is opt-in via rules.OptIn
// (DESIGN-0005 §5): without a `rule "mcp/server-allowlist"` block in
// `.claudelint.hcl` the engine skips it entirely.
//
// Behaviour matrix, once a rule block exists:
//
//	allowlist unset       → one error per server: "rule enabled without
//	                        allowlist option set" — an explicit enable
//	                        without an allowlist is a misconfiguration,
//	                        not a silent no-op
//	allowlist = []        → fires on every server (explicit "block all")
//	allowlist = ["x", …]  → fires on every server whose name is not "x"
//
// The two-step rollout (declare allowlist → restrict) lets marketplace
// owners stage rollout: add the block with a permissive allowlist,
// then tighten it to only vetted servers.
type serverAllowlist struct{}

// OptIn marks the rule disabled-by-default; the engine activates it
// only when the config contains a rule block for it.
func (*serverAllowlist) OptIn() bool { return true }

func (*serverAllowlist) ID() string                     { return "mcp/server-allowlist" }
func (*serverAllowlist) Category() string               { return categorySecurity }
func (*serverAllowlist) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*serverAllowlist) DefaultOptions() map[string]any {
	// nil declares the key so config-supplied values overlay correctly,
	// while letting Check distinguish "user didn't set it" from "user
	// set an empty list".
	return map[string]any{allowlistOption: nil}
}

func (*serverAllowlist) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (*serverAllowlist) HelpURI() string { return rules.DefaultHelpURI("mcp/server-allowlist") }

func (r *serverAllowlist) Check(ctx rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok {
		return nil
	}

	raw := ctx.Option(allowlistOption)
	if raw == nil {
		return []diag.Diagnostic{{
			RuleID:  r.ID(),
			Path:    s.Path(),
			Range:   s.NameRange,
			Message: `mcp/server-allowlist enabled without an "allowlist" option; add allowlist = [...] or set enabled = false`,
		}}
	}

	list, ok := stringSliceOption(raw)
	if !ok {
		return []diag.Diagnostic{{
			RuleID:  r.ID(),
			Path:    s.Path(),
			Range:   s.NameRange,
			Message: `mcp/server-allowlist option "allowlist" must be a list of strings`,
		}}
	}

	if slices.Contains(list, s.Name) {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    s.Path(),
		Range:   s.NameRange,
		Message: "MCP server " + quoteName(s.Name) + " is not in the configured allowlist",
	}}
}

// stringSliceOption coerces a rule-option value into a []string. HCL
// lists arrive as []any after cty conversion; this helper unwraps them
// without forcing every rule that takes a list-of-strings option to
// repeat the type-switch.
func stringSliceOption(v any) ([]string, bool) {
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
