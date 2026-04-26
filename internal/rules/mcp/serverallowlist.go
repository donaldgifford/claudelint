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
// user's configured allowlist. The rule is opt-in: with no
// `allowlist` set in `.claudelint.hcl`, the rule emits a
// configuration-error diagnostic per server explaining the missing
// option, so the user notices instead of getting a silent no-op.
//
// Behaviour matrix:
//
//	allowlist unset       → one error per server: "rule enabled without
//	                        allowlist option set"
//	allowlist = []        → fires on every server (explicit "block all")
//	allowlist = ["x", …]  → fires on every server whose name is not "x"
//
// The two-step rollout (declare allowlist → restrict) lets marketplace
// owners stage rollout: ship the rule disabled, then ship a populated
// allowlist that only exempts vetted servers.
//
// claudelint's engine has no per-rule `default-enabled = false` hook
// today, so the rule's "opt-in" framing is enforced via the loud
// missing-option diagnostic rather than load-time silence. Users who
// want the rule completely silent can disable it with
// `rule "mcp/server-allowlist" { enabled = false }`.
type serverAllowlist struct{}

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
