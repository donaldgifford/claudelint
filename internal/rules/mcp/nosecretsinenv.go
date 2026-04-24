package mcp

import (
	"fmt"
	"sort"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
	"github.com/donaldgifford/claudelint/internal/rules/security"
)

func init() { rules.Register(&noSecretsInEnv{}) }

// noSecretsInEnv errors when an MCP server's env{} contains a value
// that matches the security/secrets matcher. Committed env values
// that look like credentials are the exact scenario MCP's design
// expects you to avoid — users are meant to read secrets from the
// environment or a vault, not bake them into the manifest.
//
// The matcher is shared with rules/security/ via the narrow
// MatchesSecret export; regex tables live in exactly one place.
type noSecretsInEnv struct{}

func (*noSecretsInEnv) ID() string                     { return "mcp/no-secrets-in-env" }
func (*noSecretsInEnv) Category() string               { return categorySecurity }
func (*noSecretsInEnv) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*noSecretsInEnv) DefaultOptions() map[string]any { return nil }
func (*noSecretsInEnv) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMCPServer}
}

func (r *noSecretsInEnv) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.MCPServer)
	if !ok || len(s.Env) == 0 {
		return nil
	}
	// Sort keys so output is deterministic across runs.
	keys := make([]string, 0, len(s.Env))
	for k := range s.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out []diag.Diagnostic
	for _, k := range keys {
		if !security.MatchesSecret([]byte(s.Env[k])) {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    s.Path(),
			Range:   s.NameRange,
			Message: fmt.Sprintf("MCP server %s has a credential-looking value in env[%q] — move secrets out of the manifest", quoteName(s.Name), k),
		})
	}
	return out
}
