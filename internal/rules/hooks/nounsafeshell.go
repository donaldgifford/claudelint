package hooks

import (
	"regexp"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&noUnsafeShell{}) }

// noUnsafeShell warns when a hook command pipes an HTTP download into
// an interpreter (`curl ... | sh`, `wget ... | bash`, and variants).
// That pattern is the single most common vector for accidentally
// running arbitrary code from a network source — a warning is
// enough to surface it; users who know what they are doing can
// suppress the rule.
type noUnsafeShell struct{}

// unsafeShellPattern matches `curl|wget|fetch ... | sh|bash|zsh`
// with any flags and redirections between. The regex is intentionally
// loose so subtle variants do not slip through; false positives are
// acceptable at warning severity.
var unsafeShellPattern = regexp.MustCompile(
	`\b(curl|wget|fetch)\b[^|]*\|\s*(sh|bash|zsh|ksh|fish|dash)\b`,
)

func (*noUnsafeShell) ID() string                     { return "hooks/no-unsafe-shell" }
func (*noUnsafeShell) Category() string               { return "security" }
func (*noUnsafeShell) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*noUnsafeShell) DefaultOptions() map[string]any { return nil }
func (*noUnsafeShell) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindHook}
}

func (r *noUnsafeShell) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	h, ok := a.(*artifact.Hook)
	if !ok {
		return nil
	}
	var out []diag.Diagnostic
	for i := range h.Entries {
		e := &h.Entries[i]
		if !unsafeShellPattern.MatchString(e.Command) {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    h.Path(),
			Range:   e.CommandRange,
			Message: "hook command pipes a network download into a shell — review for supply-chain risk",
		})
	}
	return out
}
