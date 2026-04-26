package security

import (
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

func TestSecretsKnownPrefix(t *testing.T) {
	src := []byte("# tmp\n\nkey: sk-ABCDEFGHIJKLMNOPQRSTUV12345\n")
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	d := (&secrets{}).Check(nil, c)
	if len(d) == 0 {
		t.Fatal("sk- prefixed token should be flagged")
	}
}

func TestSecretsClean(t *testing.T) {
	src := []byte("# Clean\n\nNo secrets here, just prose.\n")
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	if d := (&secrets{}).Check(nil, c); len(d) != 0 {
		t.Errorf("clean file should emit nothing, got %v", d)
	}
}

func TestSecretsHighEntropy(t *testing.T) {
	// 40-character random-looking token.
	src := []byte("token: abcdefghijklmnopqrstuvwxyzABCDEFGHIJKL09\n")
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	d := (&secrets{}).Check(nil, c)
	// Entropy of this alphabetic sequence is high enough to trigger;
	// a long repeating string would not.
	if len(d) == 0 {
		t.Errorf("expected at least one diagnostic for high-entropy token")
	}
}

func TestSecretsLowEntropyTokenIgnored(t *testing.T) {
	// 40 of the same character — low entropy, should not flag.
	src := []byte("path: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n")
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	if d := (&secrets{}).Check(nil, c); len(d) != 0 {
		t.Errorf("low-entropy long token should not flag, got %v", d)
	}
}

// TestSecretsRangePointsAtMatch is the regression for issue #15: the
// rule used to emit Range == zero, making per-line suppression
// impossible because the engine could not match a marker to any line.
// The diagnostic must now point at the matched token's line + column.
func TestSecretsRangePointsAtMatch(t *testing.T) {
	// Token sits on line 3, after "key: " (5 chars), so the match
	// starts at column 6 of that line.
	src := []byte("# Title\n\nkey: sk-ABCDEFGHIJKLMNOPQRSTUV12345\n")
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	d := (&secrets{}).Check(nil, c)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
	if d[0].Range.Start.Line != 3 {
		t.Errorf("Range.Start.Line = %d, want 3", d[0].Range.Start.Line)
	}
	if d[0].Range.Start.Column != 6 {
		t.Errorf("Range.Start.Column = %d, want 6", d[0].Range.Start.Column)
	}
	if d[0].Range.End.Line != 3 {
		t.Errorf("Range.End.Line = %d, want 3", d[0].Range.End.Line)
	}
	if d[0].Range.End.Column <= d[0].Range.Start.Column {
		t.Errorf("Range.End.Column = %d, want > Start.Column %d",
			d[0].Range.End.Column, d[0].Range.Start.Column)
	}
}

// TestSecretsHighEntropyRangePointsAtMatch is the regression for the
// high-entropy code path of the rule, which previously also emitted
// Range == zero.
func TestSecretsHighEntropyRangePointsAtMatch(t *testing.T) {
	// Token sits on line 3.
	src := []byte("intro\n\ntoken: abcdefghijklmnopqrstuvwxyzABCDEFGHIJKL09\n")
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	d := (&secrets{}).Check(nil, c)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
	if d[0].Range.Start.Line != 3 {
		t.Errorf("Range.Start.Line = %d, want 3", d[0].Range.Start.Line)
	}
	if d[0].Range.IsZero() {
		t.Errorf("Range should not be zero — per-line suppression depends on it")
	}
}
