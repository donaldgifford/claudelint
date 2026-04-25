package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMarketplaceHintsMissing(t *testing.T) {
	root := t.TempDir()
	hints, err := LoadMarketplaceHints(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hints.ManifestPath != "" || len(hints.PluginRoots) != 0 {
		t.Errorf("missing manifest: want zero hints, got %+v", hints)
	}
}

func TestLoadMarketplaceHintsTraditional(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude-plugin"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `{
  "name": "x",
  "version": "1.0.0",
  "plugins": [
    {"name": "a", "source": "./plugins/a"},
    {"name": "b", "source": "./plugins/b"},
    {"name": "ext", "source": "github:owner/repo"}
  ]
}`
	manifest := filepath.Join(root, ".claude-plugin", "marketplace.json")
	if err := os.WriteFile(manifest, []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	hints, err := LoadMarketplaceHints(root)
	if err != nil {
		t.Fatalf("LoadMarketplaceHints: %v", err)
	}
	if hints.ManifestPath != ".claude-plugin/marketplace.json" {
		t.Errorf("ManifestPath = %q", hints.ManifestPath)
	}
	wantRoots := []string{"plugins/a", "plugins/b"}
	if len(hints.PluginRoots) != len(wantRoots) {
		t.Fatalf("PluginRoots = %v, want %v", hints.PluginRoots, wantRoots)
	}
	for i, want := range wantRoots {
		if hints.PluginRoots[i] != want {
			t.Errorf("PluginRoots[%d] = %q, want %q", i, hints.PluginRoots[i], want)
		}
	}
}

func TestLoadMarketplaceHintsMalformedTolerant(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude-plugin"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifest := filepath.Join(root, ".claude-plugin", "marketplace.json")
	if err := os.WriteFile(manifest, []byte(`{"name":`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	hints, err := LoadMarketplaceHints(root)
	if err != nil {
		t.Fatalf("malformed manifest should not error: %v", err)
	}
	if hints.ManifestPath != ".claude-plugin/marketplace.json" {
		t.Errorf("ManifestPath = %q", hints.ManifestPath)
	}
	if len(hints.PluginRoots) != 0 {
		t.Errorf("PluginRoots = %v, want empty on malformed", hints.PluginRoots)
	}
}
