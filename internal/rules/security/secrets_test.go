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
