package claudemd

import (
	"fmt"
	"strings"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&duplicateDirectives{}) }

// duplicateDirectives warns when a CLAUDE.md contains the same
// top-level bullet directive twice. Duplicates are almost always an
// unintentional copy-paste and they dilute the weight of genuine
// repeated advice.
//
// v1 uses a naive match: lowercase, trim whitespace and leading
// bullet characters, look for duplicates. More nuanced
// contradiction detection (what the success-criteria calls "two
// directives contradict") is deferred — it requires semantic
// understanding the static rule cannot provide.
type duplicateDirectives struct{}

func (*duplicateDirectives) ID() string                     { return "claude_md/duplicate-directives" }
func (*duplicateDirectives) Category() string               { return "content" }
func (*duplicateDirectives) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*duplicateDirectives) DefaultOptions() map[string]any { return nil }
func (*duplicateDirectives) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindClaudeMD}
}

func (r *duplicateDirectives) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	c, ok := a.(*artifact.ClaudeMD)
	if !ok {
		return nil
	}
	seen := make(map[string]int, 32)
	var out []diag.Diagnostic
	lineNo := 0
	for _, raw := range strings.Split(string(c.Source()), "\n") {
		lineNo++
		d, ok := directiveKey(raw)
		if !ok {
			continue
		}
		if first, dup := seen[d]; dup {
			out = append(out, diag.Diagnostic{
				RuleID:  r.ID(),
				Path:    c.Path(),
				Range:   diag.Range{Start: diag.Position{Line: lineNo, Column: 1}},
				Message: fmt.Sprintf("directive repeated (first seen at line %d)", first),
			})
			continue
		}
		seen[d] = lineNo
	}
	return out
}

// directiveKey extracts a canonical directive string from a CLAUDE.md
// line. Returns (key, true) only for lines that look like a bullet
// directive; other content lines are skipped so paragraph text does
// not generate noise.
func directiveKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "*") {
		return "", false
	}
	// Strip the bullet and any trailing punctuation to treat "- x"
	// and "- x." as the same directive.
	body := strings.TrimSpace(trimmed[1:])
	body = strings.TrimRight(body, ".!?")
	if body == "" {
		return "", false
	}
	return strings.ToLower(body), true
}
