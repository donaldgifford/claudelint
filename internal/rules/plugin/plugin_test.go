package plugin

import (
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// TestRuleMetadata covers the static Rule-interface methods on every
// rule in this package. Keeps the coverage gate honest and catches
// typos in IDs, severities, and applicability without having to write
// a Check-level fixture for each metadata shape.
func TestRuleMetadata(t *testing.T) {
	cases := []rules.Rule{
		&manifestFields{},
		&semverRule{},
	}
	for _, r := range cases {
		t.Run(r.ID(), func(t *testing.T) {
			if r.Category() == "" {
				t.Errorf("empty Category")
			}
			if r.DefaultSeverity() < diag.SeverityInfo || r.DefaultSeverity() > diag.SeverityError {
				t.Errorf("unexpected severity %d", r.DefaultSeverity())
			}
			// DefaultOptions may legitimately be nil; just exercise the method.
			_ = r.DefaultOptions()
			kinds := r.AppliesTo()
			if len(kinds) == 0 {
				t.Errorf("AppliesTo is empty")
			}
			for _, k := range kinds {
				if k != artifact.KindPlugin {
					t.Errorf("unexpected kind %q (plugin rules should only apply to plugin)", k)
				}
			}
			help := r.HelpURI()
			if !strings.HasPrefix(help, "https://") {
				t.Errorf("HelpURI %q is not an https URL", help)
			}
		})
	}
}

func TestManifestFieldsOK(t *testing.T) {
	src := []byte(`{"name":"x","version":"1.0.0"}`)
	p, _ := artifact.ParsePlugin("plugin.json", src)
	if d := (&manifestFields{}).Check(nil, p); len(d) != 0 {
		t.Errorf("expected no diagnostics, got %v", d)
	}
}

func TestManifestFieldsMissingName(t *testing.T) {
	src := []byte(`{"version":"1.0.0"}`)
	p, _ := artifact.ParsePlugin("plugin.json", src)
	d := (&manifestFields{}).Check(nil, p)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
}

func TestSemverValid(t *testing.T) {
	tests := []string{"1.0.0", "v1.2.3", "2.0.0-rc.1", "1.0.0+build.1"}
	for _, v := range tests {
		p := &artifact.Plugin{Version: v}
		if d := (&semverRule{}).Check(nil, p); len(d) != 0 {
			t.Errorf("valid %q: got diagnostic %v", v, d)
		}
	}
}

func TestSemverInvalid(t *testing.T) {
	tests := []string{"1", "1.0", "dev", "v1.x", "release-1"}
	for _, v := range tests {
		p := &artifact.Plugin{Version: v}
		if d := (&semverRule{}).Check(nil, p); len(d) != 1 {
			t.Errorf("invalid %q: want 1 diagnostic, got %v", v, d)
		}
	}
}
