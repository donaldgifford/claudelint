// Package security holds rules that spot dangerous patterns.
package security

import (
	"math"
	"regexp"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&secrets{}) }

// secrets errors when a file contains a string that looks like an
// API key or access token. Detection combines a shortlist of
// well-known prefixes (sk-, ghp_, AKIA, etc.) with a Shannon-entropy
// heuristic so domain-specific secrets still get flagged.
//
// The rule intentionally lives outside artifact types — it walks
// Source bytes — so it applies uniformly to every kind.
type secrets struct{}

func (*secrets) ID() string                     { return "security/secrets" }
func (*secrets) Category() string               { return "security" }
func (*secrets) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*secrets) DefaultOptions() map[string]any { return nil }
func (*secrets) AppliesTo() []artifact.ArtifactKind {
	return artifact.AllKinds()
}

var knownPrefixes = regexp.MustCompile(
	`\b(sk-[A-Za-z0-9]{20,}|` +
		`ghp_[A-Za-z0-9]{20,}|` +
		`gho_[A-Za-z0-9]{20,}|` +
		`AKIA[A-Z0-9]{16}|` +
		`AIza[A-Za-z0-9_-]{35}|` +
		`xox[baprs]-[A-Za-z0-9-]{10,}|` +
		`eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,})\b`,
)

// highEntropyCandidate matches long hex/base64 tokens that could be
// secrets. The rule combines this with an entropy check before
// flagging; most "long tokens" in Claude artifacts are path or
// config scaffolding, not credentials.
var highEntropyCandidate = regexp.MustCompile(`[A-Za-z0-9+/=]{40,}`)

const minSecretEntropy = 4.0

func (r *secrets) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	src := a.Source()
	var out []diag.Diagnostic

	for _, match := range knownPrefixes.FindAllIndex(src, -1) {
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    a.Path(),
			Message: "file contains a token matching a known API-key prefix",
		})
		_ = match
	}

	for _, match := range highEntropyCandidate.FindAllIndex(src, -1) {
		token := src[match[0]:match[1]]
		if shannonEntropy(token) < minSecretEntropy {
			continue
		}
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    a.Path(),
			Message: "file contains a high-entropy token that resembles a secret",
		})
	}
	return out
}

// MatchesSecret reports whether b contains a token that looks like a
// credential. It is the matcher the secrets rule uses, exposed for
// other rule packages (e.g. rules/mcp) that want to check field-level
// values without reimplementing the pattern bundle. The decision
// combines the known-prefix regex with the Shannon-entropy filter
// over the high-entropy candidate regex.
//
// This is the narrow public API the security package exposes on
// purpose — larger matcher consumers should live inside this
// package or move to a shared helper.
func MatchesSecret(b []byte) bool {
	if knownPrefixes.Match(b) {
		return true
	}
	for _, match := range highEntropyCandidate.FindAllIndex(b, -1) {
		token := b[match[0]:match[1]]
		if shannonEntropy(token) >= minSecretEntropy {
			return true
		}
	}
	return false
}

// shannonEntropy computes the Shannon entropy (in bits per symbol)
// of b. Used to distinguish genuinely random tokens from repetitive
// or structured strings.
func shannonEntropy(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}
	counts := [256]int{}
	for _, c := range b {
		counts[c]++
	}
	var ent float64
	n := float64(len(b))
	for i := range counts {
		if counts[i] == 0 {
			continue
		}
		p := float64(counts[i]) / n
		ent -= p * math.Log2(p)
	}
	return ent
}
