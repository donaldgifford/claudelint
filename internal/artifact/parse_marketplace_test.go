package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseMarketplaceFixtures exercises the three canonical layouts
// against parser-level invariants (name/version resolution, plugins
// array length, Resolved paths).
func TestParseMarketplaceFixtures(t *testing.T) {
	type wantPlugin struct {
		name, source, resolved string
	}

	tests := []struct {
		name     string
		file     string
		relPath  string
		wantName string
		wantVer  string
		wantAuth string
		plugins  []wantPlugin
	}{
		{
			name:     "flat",
			file:     "testdata/ok/marketplaces/flat/marketplace.json",
			relPath:  "my-plugin/.claude-plugin/marketplace.json",
			wantName: "claude-skills",
			wantVer:  "1.0.0",
			wantAuth: "Donald Gifford",
			plugins: []wantPlugin{
				{name: "claude-skills", source: "./", resolved: "my-plugin"},
			},
		},
		{
			name:     "traditional",
			file:     "testdata/ok/marketplaces/traditional/marketplace.json",
			relPath:  "repo/.claude-plugin/marketplace.json",
			wantName: "anthropic-plugins",
			wantVer:  "2.3.1",
			wantAuth: "Anthropic",
			plugins: []wantPlugin{
				{name: "donald-loop", source: "./plugins/donald-loop", resolved: "repo/plugins/donald-loop"},
				{name: "docz", source: "./plugins/docz", resolved: "repo/plugins/docz"},
			},
		},
		{
			name:     "mixed-external",
			file:     "testdata/ok/marketplaces/mixed/marketplace.json",
			relPath:  ".claude-plugin/marketplace.json",
			wantName: "mixed",
			wantVer:  "0.0.1",
			plugins: []wantPlugin{
				{name: "local-one", source: "./plugins/local-one", resolved: "plugins/local-one"},
				{name: "external-git", source: "github:acme/cool-plugin", resolved: ""},
				{name: "external-https", source: "https://example.com/some/repo.git", resolved: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			m, perr := ParseMarketplace(tt.relPath, src)
			if perr != nil {
				t.Fatalf("ParseMarketplace error: %v", perr)
			}
			if m.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", m.Name, tt.wantName)
			}
			if m.Version != tt.wantVer {
				t.Errorf("Version = %q, want %q", m.Version, tt.wantVer)
			}
			if m.Author != tt.wantAuth {
				t.Errorf("Author = %q, want %q", m.Author, tt.wantAuth)
			}
			if got, want := len(m.Plugins), len(tt.plugins); got != want {
				t.Fatalf("len(Plugins) = %d, want %d", got, want)
			}
			for i, want := range tt.plugins {
				got := m.Plugins[i]
				if got.Name != want.name {
					t.Errorf("Plugins[%d].Name = %q, want %q", i, got.Name, want.name)
				}
				if got.Source != want.source {
					t.Errorf("Plugins[%d].Source = %q, want %q", i, got.Source, want.source)
				}
				if got.Resolved != want.resolved {
					t.Errorf("Plugins[%d].Resolved = %q, want %q", i, got.Resolved, want.resolved)
				}
			}
		})
	}
}

// TestParseMarketplaceBadFixtures verifies the parser's error paths
// without asserting message text — messages are allowed to drift, but
// the presence of a ParseError (and its Range landing inside the
// file) is contractual.
func TestParseMarketplaceBadFixtures(t *testing.T) {
	tests := []struct {
		name   string
		file   string
		nonNil bool
	}{
		{"malformed", "testdata/bad/marketplace_malformed.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, err := os.ReadFile(tt.file)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			_, perr := ParseMarketplace(".claude-plugin/marketplace.json", src)
			if tt.nonNil && perr == nil {
				t.Fatalf("expected ParseError, got nil")
			}
		})
	}
}

// TestParseMarketplaceTolerant covers the lenient shapes: a missing
// plugins field yields an empty slice (not an error), and a non-string
// source inside an entry yields an empty Source for that entry while
// the rest of the array still parses.
func TestParseMarketplaceTolerant(t *testing.T) {
	t.Run("missing plugins field", func(t *testing.T) {
		src, err := os.ReadFile("testdata/bad/marketplace_missing_plugins.json")
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		m, perr := ParseMarketplace(".claude-plugin/marketplace.json", src)
		if perr != nil {
			t.Fatalf("ParseMarketplace error: %v", perr)
		}
		if len(m.Plugins) != 0 {
			t.Errorf("Plugins = %+v, want empty", m.Plugins)
		}
	})

	t.Run("non-string source", func(t *testing.T) {
		src, err := os.ReadFile("testdata/bad/marketplace_nonstring_source.json")
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		m, perr := ParseMarketplace(".claude-plugin/marketplace.json", src)
		if perr != nil {
			t.Fatalf("ParseMarketplace error: %v", perr)
		}
		if len(m.Plugins) != 1 {
			t.Fatalf("len(Plugins) = %d, want 1", len(m.Plugins))
		}
		if m.Plugins[0].Source != "" {
			t.Errorf("Source = %q, want empty (non-string value skipped)", m.Plugins[0].Source)
		}
	})
}

// TestParseMarketplaceSourceRange verifies the SourceRange for a known
// fixture points at a byte span whose contents, when sliced out of
// Raw/Source, match the expected source string.
func TestParseMarketplaceSourceRange(t *testing.T) {
	src, err := os.ReadFile("testdata/ok/marketplaces/traditional/marketplace.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	m, perr := ParseMarketplace(".claude-plugin/marketplace.json", src)
	if perr != nil {
		t.Fatalf("ParseMarketplace error: %v", perr)
	}
	if len(m.Plugins) == 0 {
		t.Fatal("no plugins parsed")
	}
	first := m.Plugins[0]
	if first.SourceRange.Start.Offset == 0 && first.SourceRange.End.Offset == 0 {
		t.Fatal("SourceRange is zero")
	}
	snippet := string(m.Source()[first.SourceRange.Start.Offset:first.SourceRange.End.Offset])
	// Snippet includes surrounding quotes; trim them for comparison.
	snippet = strings.Trim(snippet, `"`)
	if snippet != first.Source {
		t.Errorf("SourceRange snippet = %q, want %q", snippet, first.Source)
	}
}

// TestMarketplaceRoot covers the edge cases of directory resolution so
// the parser's implicit contract (dir strip) is regression-guarded.
func TestMarketplaceRoot(t *testing.T) {
	tests := []struct {
		path, want string
	}{
		{".claude-plugin/marketplace.json", "."},
		{"sub/.claude-plugin/marketplace.json", "sub"},
		{filepath.ToSlash("deep/nest/.claude-plugin/marketplace.json"), "deep/nest"},
	}
	for _, tt := range tests {
		if got := marketplaceRoot(tt.path); got != tt.want {
			t.Errorf("marketplaceRoot(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
