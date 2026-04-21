// Package style holds cosmetic rules. These are off by default so
// teams with strong style preferences opt in via config rather than
// fighting rules out of the box.
package style

import (
	"unicode"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&noEmoji{}) }

// noEmoji is an info-severity rule that spots emoji in
// output-influencing text (descriptions, allowed-tools labels, hook
// commands). It is purely advisory — the DefaultSeverity is Info,
// which does not fail runs, and users can disable it entirely via
// config. The rule applies to every artifact kind so teams that
// want emoji-free docs can adopt it wholesale.
type noEmoji struct{}

func (*noEmoji) ID() string                     { return "style/no-emoji" }
func (*noEmoji) Category() string               { return "style" }
func (*noEmoji) DefaultSeverity() diag.Severity { return diag.SeverityInfo }
func (*noEmoji) DefaultOptions() map[string]any { return nil }
func (*noEmoji) AppliesTo() []artifact.ArtifactKind {
	return artifact.AllKinds()
}

func (r *noEmoji) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	if !containsEmoji(a.Source()) {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    a.Path(),
		Message: "artifact contains emoji (style/no-emoji is advisory)",
	}}
}

// containsEmoji reports whether b contains a rune that the stdlib's
// Unicode tables tag as a "symbol" — a loose, fast proxy for emoji.
// False positives (Greek letters, mathematical symbols) are
// acceptable because the rule is info-only.
func containsEmoji(b []byte) bool {
	for _, r := range string(b) {
		if r < 128 {
			continue
		}
		if unicode.Is(unicode.So, r) {
			return true
		}
	}
	return false
}
