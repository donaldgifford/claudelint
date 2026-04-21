package plugin

import (
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

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
