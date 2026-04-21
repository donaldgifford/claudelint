// Package skills holds rules that only apply to Skill artifacts.
package skills

import (
	"bytes"
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&bodySize{}) }

const defaultMaxWords = 1000

// bodySize warns when a skill's body exceeds a configurable word
// count. Large skills dilute the trigger phrase and push the skill
// toward "implicit documentation" territory — the rule surfaces
// that growth so authors notice before it snowballs.
//
// Options:
//
//	max_words (int): maximum allowed body word count, default 1000.
type bodySize struct{}

func (*bodySize) ID() string                     { return "skills/body-size" }
func (*bodySize) Category() string               { return "content" }
func (*bodySize) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*bodySize) DefaultOptions() map[string]any {
	return map[string]any{"max_words": defaultMaxWords}
}

func (*bodySize) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindSkill}
}

func (r *bodySize) Check(ctx rules.Context, a artifact.Artifact) []diag.Diagnostic {
	s, ok := a.(*artifact.Skill)
	if !ok {
		return nil
	}
	maxWords := intOption(ctx, "max_words", defaultMaxWords)
	words := countWords(s.Source()[s.Body.Start.Offset:s.Body.End.Offset])
	if words <= maxWords {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    s.Path(),
		Range:   s.Body,
		Message: fmt.Sprintf("SKILL.md body has %d words; max is %d", words, maxWords),
	}}
}

// countWords returns the number of whitespace-separated tokens in b.
// It is intentionally conservative: CommonMark structure, code
// fences, and comments all count. Rules author's job is to advise,
// not to second-guess what counts as "content".
func countWords(b []byte) int {
	if len(bytes.TrimSpace(b)) == 0 {
		return 0
	}
	count := 0
	inWord := false
	for _, c := range b {
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if inWord {
				count++
				inWord = false
			}
			continue
		}
		inWord = true
	}
	if inWord {
		count++
	}
	return count
}

// intOption coerces a rule option to an int, falling back to def on
// any mismatch. Default values come in as int via DefaultOptions;
// config-provided values may come in as int64 via cty conversion.
func intOption(ctx rules.Context, key string, def int) int {
	switch v := ctx.Option(key).(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return def
	}
}
