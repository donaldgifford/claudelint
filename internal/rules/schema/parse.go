// Package schema holds rules whose domain is the schema/shape of an
// artifact — frontmatter completeness, parseability, required
// fields. These rules apply across every artifact kind.
package schema

import (
	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&parseRule{}) }

// parseRule is the pseudo-rule that claims the schema/parse ID in the
// registry. Its Check is never called — the engine synthesizes the
// diagnostic directly from *artifact.ParseError during Run. Existing
// in the registry lets users disable or adjust parse-error severity
// via config just like any other rule and makes `claudelint rules`
// enumerate parse errors as a first-class rule.
type parseRule struct{}

func (*parseRule) ID() string                     { return "schema/parse" }
func (*parseRule) Category() string               { return "schema" }
func (*parseRule) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*parseRule) DefaultOptions() map[string]any { return nil }
func (*parseRule) AppliesTo() []artifact.ArtifactKind {
	return artifact.AllKinds()
}

// Check is a no-op. Engine.Run never dispatches to it; it exists only
// to satisfy the Rule interface.

func (*parseRule) HelpURI() string { return rules.DefaultHelpURI("schema/parse") }

func (*parseRule) Check(_ rules.Context, _ artifact.Artifact) []diag.Diagnostic {
	return nil
}
