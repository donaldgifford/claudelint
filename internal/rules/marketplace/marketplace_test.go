package marketplace

import (
	"strings"
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
		{"missing is version-missing's job", "", 0},
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

func TestVersionMissing(t *testing.T) {
	missing := &artifact.Marketplace{}
	if d := (&versionMissing{}).Check(nil, missing); len(d) != 1 {
		t.Errorf("missing version: want 1 info, got %d", len(d))
	}
	declared := &artifact.Marketplace{Version: "dev"}
	if d := (&versionMissing{}).Check(nil, declared); len(d) != 0 {
		t.Errorf("declared version (even malformed): want 0, got %v", d)
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

func TestPluginSourceValidObjectSources(t *testing.T) {
	const goodSHA = "0123456789abcdef0123456789abcdef01234567"
	cases := []struct {
		name   string
		source string
		wantN  int
	}{
		{"github ok", `{"source":"github","repo":"owner/repo"}`, 0},
		{"github missing repo", `{"source":"github"}`, 1},
		{"url ok", `{"source":"url","url":"https://git.example.com/x.git"}`, 0},
		{"url missing url", `{"source":"url"}`, 1},
		{"git-subdir ok", `{"source":"git-subdir","url":"https://git.example.com/x.git","path":"plugins/a"}`, 0},
		{"git-subdir missing both", `{"source":"git-subdir"}`, 2},
		{"npm ok", `{"source":"npm","package":"@scope/name"}`, 0},
		{"npm missing package", `{"source":"npm"}`, 1},
		{"unknown discriminator", `{"source":"carrier-pigeon"}`, 1},
		{"github with full sha", `{"source":"github","repo":"o/r","sha":"` + goodSHA + `"}`, 0},
		{"github with short sha", `{"source":"github","repo":"o/r","sha":"abc123"}`, 1},
		{"url with non-hex sha", `{"source":"url","url":"https://x","sha":"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"}`, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMarketplace(t,
				`{"name":"m","version":"1.0.0","plugins":[{"name":"p","source":`+tc.source+`}]}`)
			d := (&pluginSourceValid{}).Check(nil, m)
			if len(d) != tc.wantN {
				t.Errorf("got %d diagnostics, want %d (%v)", len(d), tc.wantN, d)
			}
		})
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

func TestOwnerRequiredAndAuthorLegacy(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantOwner  int // diagnostics from marketplace/owner-required
		wantLegacy int // diagnostics from marketplace/author-legacy
	}{
		{
			"neither owner nor author",
			`{"name":"m","plugins":[]}`,
			1, 0,
		},
		{
			"owner object",
			`{"name":"m","owner":{"name":"Donald","email":"d@example.com"},"plugins":[]}`,
			0, 0,
		},
		{
			"legacy author only",
			`{"name":"m","author":"Donald","plugins":[]}`,
			0, 1,
		},
		{
			"both owner and author",
			`{"name":"m","author":"Donald","owner":{"name":"Donald"},"plugins":[]}`,
			0, 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMarketplace(t, tc.body)
			if d := (&ownerRequired{}).Check(nil, m); len(d) != tc.wantOwner {
				t.Errorf("owner-required: want %d, got %d (%v)", tc.wantOwner, len(d), d)
			}
			if d := (&authorLegacy{}).Check(nil, m); len(d) != tc.wantLegacy {
				t.Errorf("author-legacy: want %d, got %d (%v)", tc.wantLegacy, len(d), d)
			}
		})
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

func TestExternalSourceSkippedObjectSources(t *testing.T) {
	m := newMarketplace(t, `{"name":"m","version":"1.0.0","plugins":[
		{"name":"local","source":"./plugins/local"},
		{"name":"gh","source":{"source":"github","repo":"o/r"}},
		{"name":"web","source":{"source":"url","url":"https://git.example.com/x.git"}},
		{"name":"sub","source":{"source":"git-subdir","url":"https://git.example.com/x.git","path":"p"}},
		{"name":"pkg","source":{"source":"npm","package":"@scope/name"}},
		{"name":"bogus","source":{"source":"carrier-pigeon"}}
	]}`)
	d := (&externalSourceSkipped{}).Check(nil, m)
	// Four remote object kinds flag; local and invalid do not (invalid
	// is plugin-source-valid's finding, not a skip).
	if len(d) != 4 {
		t.Fatalf("want 4 info diagnostics, got %d (%v)", len(d), d)
	}
	for _, dd := range d {
		if !strings.Contains(dd.Message, "remote") {
			t.Errorf("message should say remote, got %q", dd.Message)
		}
	}
}

func TestReservedName(t *testing.T) {
	ok := newMarketplace(t,
		`{"name":"acme-tools","owner":{"name":"acme"},"plugins":[]}`)
	if d := (&reservedName{}).Check(nil, ok); len(d) != 0 {
		t.Errorf("ordinary name flagged: %v", d)
	}

	for _, bad := range []string{"anthropic-plugins", "healthcare", "claude-code-marketplace"} {
		m := newMarketplace(t,
			`{"name":"`+bad+`","owner":{"name":"acme"},"plugins":[]}`)
		d := (&reservedName{}).Check(nil, m)
		if len(d) != 1 {
			t.Fatalf("%s: got %d diagnostics, want 1 (%v)", bad, len(d), d)
		}
		if !strings.Contains(d[0].Message, "reserved for official Anthropic use") {
			t.Errorf("message = %q", d[0].Message)
		}
		if d[0].Range.IsZero() {
			t.Errorf("diagnostic should anchor at the name key")
		}
	}

	// Near-misses are server-side territory, not ours.
	near := newMarketplace(t,
		`{"name":"official-claude-plugins","owner":{"name":"acme"},"plugins":[]}`)
	if d := (&reservedName{}).Check(nil, near); len(d) != 0 {
		t.Errorf("impersonation heuristics should not fire locally: %v", d)
	}
}
