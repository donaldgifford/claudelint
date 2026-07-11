package schema

import (
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&frontmatterRequired{}) }

// frontmatterRequired enforces that skill, command, and agent
// artifacts carry non-empty `name` and `description` frontmatter
// keys. Missing keys land on the whole frontmatter block; empty
// values land on the offending key's line.
type frontmatterRequired struct{}

func (*frontmatterRequired) ID() string                     { return "schema/frontmatter-required" }
func (*frontmatterRequired) Category() string               { return "schema" }
func (*frontmatterRequired) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*frontmatterRequired) DefaultOptions() map[string]any { return nil }
func (*frontmatterRequired) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{
		artifact.KindSkill,
		artifact.KindCommand,
		artifact.KindAgent,
	}
}

func (*frontmatterRequired) HelpURI() string {
	return rules.DefaultHelpURI("schema/frontmatter-required")
}

func (r *frontmatterRequired) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	fm, name, description := extractFrontmatterFields(a)

	// A command/skill/agent always has frontmatter in practice;
	// check name first unless Block is zero (no frontmatter), in
	// which case both keys are effectively missing.
	var out []diag.Diagnostic
	if fm.Block.IsZero() {
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    a.Path(),
			Message: "artifact is missing required YAML frontmatter",
		})
		return out
	}
	if name == "" {
		out = append(out, missingKeyDiag(r.ID(), a.Path(), fm, "name", keyMessage(a, "name")))
	}
	if description == "" {
		out = append(out, missingKeyDiag(r.ID(), a.Path(), fm, "description", keyMessage(a, "description")))
	}
	return out
}

// keyMessage phrases the missing-key diagnostic. Skills get the
// best-practice form citing Claude Code's documented fallback — the
// docs don't hard-require these keys, so their requiredness here is
// claudelint's stricter-than-spec stance (see rules.md). Other kinds
// keep the plain form.
func keyMessage(a artifact.Artifact, key string) string {
	if _, ok := a.(*artifact.Skill); ok {
		switch key {
		case "name":
			return `skills should declare "name" — without it Claude Code falls back to the skill directory name`
		case "description":
			return `skills should declare "description" — Claude Code relies on it to decide when to invoke the skill`
		}
	}
	return fmt.Sprintf("frontmatter key %q is missing or empty", key)
}

// extractFrontmatterFields pulls Frontmatter, name, and description
// off the three Markdown artifact types. Returns zero values when the
// type assertion fails, which cannot happen through a correctly-
// routed engine dispatch.
func extractFrontmatterFields(a artifact.Artifact) (artifact.Frontmatter, string, string) {
	switch v := a.(type) {
	case *artifact.Skill:
		return v.Frontmatter, v.Name, v.Description
	case *artifact.Command:
		// Commands don't have a "name" — the filename is the name —
		// so we only check description.
		return v.Frontmatter, "n/a", v.Description
	case *artifact.Agent:
		return v.Frontmatter, v.Name, v.Description
	}
	return artifact.Frontmatter{}, "", ""
}

func missingKeyDiag(ruleID, path string, fm artifact.Frontmatter, key, message string) diag.Diagnostic {
	r := fm.KeyRange(key)
	if r.IsZero() {
		// Key is entirely absent — point at the opening frontmatter
		// fence so users see the block that needs the key.
		r = fm.Block
	}
	return diag.Diagnostic{
		RuleID:  ruleID,
		Path:    path,
		Range:   r,
		Message: message,
	}
}
