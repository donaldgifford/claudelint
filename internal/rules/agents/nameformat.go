package agents

import (
	"fmt"
	"regexp"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&nameFormat{}) }

// agentNamePattern is the documented constraint: lowercase letters
// and hyphens, no leading/trailing/double hyphens.
var agentNamePattern = regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)

// nameFormat warns when an agent's name breaks the documented
// lowercase-letters-and-hyphens constraint. Hooks receive the value
// as agent_type and duplicate resolution across scopes is by
// undocumented filesystem order, so nonconforming names misbehave in
// hard-to-debug ways. Empty names are schema/frontmatter-required's
// job.
type nameFormat struct{}

func (*nameFormat) ID() string                     { return "agents/name-format" }
func (*nameFormat) Category() string               { return categorySchema }
func (*nameFormat) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*nameFormat) DefaultOptions() map[string]any { return nil }
func (*nameFormat) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindAgent}
}

func (*nameFormat) HelpURI() string { return rules.DefaultHelpURI("agents/name-format") }

func (r *nameFormat) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	ag, ok := a.(*artifact.Agent)
	if !ok || ag.Name == "" || agentNamePattern.MatchString(ag.Name) {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    ag.Path(),
		Range:   ag.Frontmatter.KeyRange("name"),
		Message: fmt.Sprintf("agent name %q should be lowercase letters and hyphens", ag.Name),
	}}
}
