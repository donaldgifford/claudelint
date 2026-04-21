package style

import (
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

func TestNoEmojiClean(t *testing.T) {
	src := []byte("# Clean\n\nNo emoji here.\n")
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	if d := (&noEmoji{}).Check(nil, c); len(d) != 0 {
		t.Errorf("clean file should emit nothing, got %v", d)
	}
}

func TestNoEmojiDirty(t *testing.T) {
	src := []byte("# Ship it 🚀\n")
	c, _ := artifact.ParseClaudeMD("CLAUDE.md", src)
	d := (&noEmoji{}).Check(nil, c)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic for emoji content, got %d", len(d))
	}
}
