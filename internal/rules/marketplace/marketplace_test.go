package marketplace

import (
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// newMarketplace builds a *artifact.Marketplace from a JSON body so
// rule tests can focus on the diagnostic side without restating the
// parse-time setup every time.
func newMarketplace(t *testing.T, body string) *artifact.Marketplace {
	t.Helper()
	m, perr := artifact.ParseMarketplace(".claude-plugin/marketplace.json", []byte(body))
	if perr != nil {
		t.Fatalf("parse: %v", perr)
	}
	return m
}

func TestNameMissing(t *testing.T) {
	m := newMarketplace(t, `{"version":"1.0.0","plugins":[]}`)
	d := (&name{}).Check(nil, m)
	if len(d) != 1 {
		t.Fatalf("want 1 diagnostic, got %d (%v)", len(d), d)
	}
}

func TestNameOK(t *testing.T) {
	m := newMarketplace(t, `{"name":"x","version":"1.0.0","plugins":[]}`)
	if d := (&name{}).Check(nil, m); len(d) != 0 {
		t.Errorf("want 0, got %v", d)
	}
}

func TestVersionSemver(t *testing.T) {
	cases := []struct {
		name    string
		version string
		wantN   int
	}{
		{"valid 1.0.0", "1.0.0", 0},
		{"valid v2.3.4", "v2.3.4", 0},
		{"missing", "", 1},
		{"not semver", "dev", 1},
		{"partial", "1.2", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &artifact.Marketplace{Version: tc.version}
			d := (&versionSemver{}).Check(nil, m)
			if len(d) != tc.wantN {
				t.Errorf("want %d diagnostics, got %d (%v)", tc.wantN, len(d), d)
			}
		})
	}
}

func TestPluginsNonempty(t *testing.T) {
	empty := &artifact.Marketplace{}
	if d := (&pluginsNonempty{}).Check(nil, empty); len(d) != 1 {
		t.Errorf("empty plugins: want 1, got %d", len(d))
	}
	full := &artifact.Marketplace{Plugins: []artifact.MarketplacePlugin{{Name: "one"}}}
	if d := (&pluginsNonempty{}).Check(nil, full); len(d) != 0 {
		t.Errorf("populated: want 0, got %v", d)
	}
}

func TestPluginSourceValid(t *testing.T) {
	m := &artifact.Marketplace{
		Plugins: []artifact.MarketplacePlugin{
			{Name: "ok", Source: "./a"},
			{Name: "missing-source"},
			{Name: "also-missing", Source: ""},
		},
	}
	d := (&pluginSourceValid{}).Check(nil, m)
	if len(d) != 2 {
		t.Errorf("want 2 diagnostics, got %d (%v)", len(d), d)
	}
}

func TestPluginNameUniqueDuplicates(t *testing.T) {
	m := &artifact.Marketplace{
		Plugins: []artifact.MarketplacePlugin{
			{Name: "one", Source: "./a"},
			{Name: "two", Source: "./b"},
			{Name: "one", Source: "./c"},
		},
	}
	d := (&pluginNameUnique{}).Check(nil, m)
	if len(d) != 1 {
		t.Errorf("want 1 diagnostic on duplicate, got %d (%v)", len(d), d)
	}
}

func TestPluginNameUniqueAllUnique(t *testing.T) {
	m := &artifact.Marketplace{
		Plugins: []artifact.MarketplacePlugin{
			{Name: "one", Source: "./a"},
			{Name: "two", Source: "./b"},
		},
	}
	if d := (&pluginNameUnique{}).Check(nil, m); len(d) != 0 {
		t.Errorf("want 0, got %v", d)
	}
}

func TestPluginNameMatchesDir(t *testing.T) {
	cases := []struct {
		name                 string
		pluginName, resolved string
		wantN                int
	}{
		{"matches", "donald-loop", "plugins/donald-loop", 0},
		{"mismatch", "donald-loop", "plugins/other-name", 1},
		{"flat root (basename == '.')", "whatever", ".", 0},
		{"external (empty resolved) skipped", "remote", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &artifact.Marketplace{Plugins: []artifact.MarketplacePlugin{{
				Name:     tc.pluginName,
				Resolved: tc.resolved,
			}}}
			d := (&pluginNameMatchesDir{}).Check(nil, m)
			if len(d) != tc.wantN {
				t.Errorf("want %d, got %d (%v)", tc.wantN, len(d), d)
			}
		})
	}
}

func TestAuthorRequired(t *testing.T) {
	missing := &artifact.Marketplace{}
	if d := (&authorRequired{}).Check(nil, missing); len(d) != 1 {
		t.Errorf("missing author: want 1, got %d", len(d))
	}
	present := &artifact.Marketplace{Author: "someone"}
	if d := (&authorRequired{}).Check(nil, present); len(d) != 0 {
		t.Errorf("present author: want 0, got %v", d)
	}
}

func TestExternalSourceSkipped(t *testing.T) {
	m := &artifact.Marketplace{
		Plugins: []artifact.MarketplacePlugin{
			{Name: "local", Source: "./a", Resolved: "a"},
			{Name: "gh", Source: "github:x/y", Resolved: ""},
			{Name: "https", Source: "https://host/repo", Resolved: ""},
			{Name: "missing-source", Source: "", Resolved: ""}, // skipped: empty source
		},
	}
	d := (&externalSourceSkipped{}).Check(nil, m)
	if len(d) != 2 {
		t.Errorf("want 2 info diagnostics, got %d (%v)", len(d), d)
	}
}
