package skills

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&forkAgentPairing{}) }

// forkAgentPairing warns when a skill declares `agent:` without
// `context: fork`. The agent field only means something for forked
// skills — it names the subagent type the fork runs as. Without the
// pairing, the declaration is dead config that reads as if the skill
// delegates to that agent.
type forkAgentPairing struct{}

func (*forkAgentPairing) ID() string                     { return "skills/fork-agent-pairing" }
func (*forkAgentPairing) Category() string               { return "schema" }
func (*forkAgentPairing) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*forkAgentPairing) DefaultOptions() map[string]any { return nil }
func (*forkAgentPairing) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindSkill}
}

func (*forkAgentPairing) HelpURI() string {
	return rules.DefaultHelpURI("skills/fork-agent-pairing")
}

func (r *forkAgentPairing) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.Skill)
	if !ok || s.Agent == "" || s.Context == "fork" {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID: r.ID(),
		Path:   s.Path(),
		Range:  s.Frontmatter.KeyRange("agent"),
		Message: fmt.Sprintf(
			"agent: %q has no effect without context: fork — dead config", s.Agent),
	}}
}
