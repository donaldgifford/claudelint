package upstream_test

import (
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

func TestSourcesTableShape(t *testing.T) {
	t.Parallel()

	srcs := upstream.Sources()
	if len(srcs) == 0 {
		t.Fatal("Sources() returned no sources")
	}

	seen := make(map[string]struct{}, len(srcs))
	tiers := make(map[byte]int)

	for _, s := range srcs {
		if _, dup := seen[s.ID]; dup {
			t.Errorf("duplicate source id %q", s.ID)
		}
		seen[s.ID] = struct{}{}

		u, err := url.Parse(s.URL)
		if err != nil {
			t.Errorf("source %s: parse url: %v", s.ID, err)

			continue
		}
		if u.Scheme != "https" {
			t.Errorf("source %s: scheme is %q, want https", s.ID, u.Scheme)
		}
		if s.Ext == "" || strings.HasPrefix(s.Ext, ".") {
			t.Errorf("source %s: ext is %q, want a bare extension", s.ID, s.Ext)
		}
		if want := s.ID + "." + s.Ext; s.File() != want {
			t.Errorf("source %s: File() = %q, want %q", s.ID, s.File(), want)
		}

		switch s.Tier {
		case upstream.TierAuthoritative, upstream.TierCrossCheck,
			upstream.TierPortable, upstream.TierMetadata:
			tiers[s.Tier]++
		default:
			t.Errorf("source %s: unknown tier %q", s.ID, s.Tier)
		}
	}

	for _, tier := range []byte{
		upstream.TierAuthoritative, upstream.TierCrossCheck,
		upstream.TierPortable, upstream.TierMetadata,
	} {
		if tiers[tier] == 0 {
			t.Errorf("no sources in tier %q", tier)
		}
	}
}

func TestSourcesSortedByID(t *testing.T) {
	t.Parallel()

	srcs := upstream.Sources()
	ids := make([]string, len(srcs))

	for i, s := range srcs {
		ids[i] = s.ID
	}

	if !sort.StringsAreSorted(ids) {
		t.Errorf("source table is not sorted by id: %v", ids)
	}
}

func TestSourcesReturnsACopy(t *testing.T) {
	t.Parallel()

	first := upstream.Sources()
	first[0].ID = "mutated"

	if got := upstream.Sources()[0].ID; got == "mutated" {
		t.Error("Sources() exposed the package table; callers can mutate it")
	}
}

func TestSourceByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		id       string
		wantOK   bool
		wantTier byte
	}{
		{name: "authoritative doc", id: upstream.SrcDocsHooks, wantOK: true, wantTier: upstream.TierAuthoritative},
		{name: "cross-check schema", id: upstream.SrcSchemaStoreSettings, wantOK: true, wantTier: upstream.TierCrossCheck},
		{name: "portable spec", id: upstream.SrcAgentSkillsSpec, wantOK: true, wantTier: upstream.TierPortable},
		{name: "metadata", id: upstream.SrcClaudeCodeChangelog, wantOK: true, wantTier: upstream.TierMetadata},
		{name: "unknown", id: "docs.nonexistent", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := upstream.SourceByID(tt.id)
			if ok != tt.wantOK {
				t.Fatalf("SourceByID(%q) ok = %v, want %v", tt.id, ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if got.ID != tt.id {
				t.Errorf("SourceByID(%q).ID = %q", tt.id, got.ID)
			}
			if got.Tier != tt.wantTier {
				t.Errorf("SourceByID(%q).Tier = %q, want %q", tt.id, got.Tier, tt.wantTier)
			}
		})
	}
}

// TestProbesAreOptional pins DESIGN-0006's rule that the
// code.claude.com/schemas probes are recorded but never fail a run.
func TestProbesAreOptional(t *testing.T) {
	t.Parallel()

	probes := []string{upstream.SrcProbePluginSchema, upstream.SrcProbeMarketplaceSchema}
	for _, id := range probes {
		s, ok := upstream.SourceByID(id)
		if !ok {
			t.Fatalf("probe %q missing from the source table", id)
		}
		if !s.Optional {
			t.Errorf("probe %q is not optional", id)
		}
	}

	for _, s := range upstream.Sources() {
		if s.Optional && !strings.HasPrefix(s.ID, "probe.") {
			t.Errorf("source %q is optional but is not a probe", s.ID)
		}
	}
}
