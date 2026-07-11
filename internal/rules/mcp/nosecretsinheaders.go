package mcp

import (
	"fmt"
	"sort"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
	"github.com/donaldgifford/claudelint/internal/rules/security"
)

func init() { rules.Register(&noSecretsInHeaders{}) }

// noSecretsInHeaders errors when a remote server's headers{} contains
// a value matching the security/secrets matcher. Kept as a separate
// rule rather than folded into mcp/no-secrets-in-env so the two
// surfaces can be suppressed independently; both reuse the shared
// security.MatchesSecret detector, so the regex tables still live in
// exactly one place. Placeholder values (`Bearer ${API_KEY}`) do not
// match — only credential-looking literals do.
type noSecretsInHeaders struct{}

func (*noSecretsInHeaders) ID() string                     { return "mcp/no-secrets-in-headers" }
func (*noSecretsInHeaders) Category() string               { return categorySecurity }
func (*noSecretsInHeaders) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*noSecretsInHeaders) DefaultOptions() map[string]any { return nil }
func (*noSecretsInHeaders) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (*noSecretsInHeaders) HelpURI() string {
	return rules.DefaultHelpURI("mcp/no-secrets-in-headers")
}

func (r *noSecretsInHeaders) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok || len(s.Headers) == 0 {
		return nil
	}
	// Sort keys so output is deterministic across runs.
	keys := make([]string, 0, len(s.Headers))
	for k := range s.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out []diag.Diagnostic
	for _, k := range keys {
		if !security.MatchesSecret([]byte(s.Headers[k])) {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID: r.ID(),
			Path:   s.Path(),
			Range:  s.NameRange,
			Message: fmt.Sprintf(
				"MCP server %s has a credential-looking value in headers[%q] — use headersHelper or an environment reference",
				quoteName(s.Name), k),
		})
	}
	return out
}
