package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfileSessionWritesAllFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "prof")

	sess, err := startProfileSession(dir)
	if err != nil {
		t.Fatalf("startProfileSession: %v", err)
	}
	// Do a tiny bit of work so the cpu profile has samples.
	sum := 0
	for i := 0; i < 100000; i++ {
		sum += i
	}
	_ = sum
	if err := sess.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	for _, name := range []string{"cpu.pprof", "heap.pprof", "block.pprof", "mutex.pprof"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("stat %s: %v", name, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestProfileSessionDoubleCloseIsNoop(t *testing.T) {
	dir := t.TempDir()

	sess, err := startProfileSession(dir)
	if err != nil {
		t.Fatalf("startProfileSession: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Fatalf("Close #1: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Errorf("double Close should be a noop, got %v", err)
	}
}
