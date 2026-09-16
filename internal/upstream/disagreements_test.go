package upstream_test

import (
	"slices"
	"testing"

	"github.com/donaldgifford/claudelint/internal/upstream"
)

// TestCrossCheckFromFixtures pins the disagreements the committed
// fixtures produce. Three of the six compared pairs differ today; the
// hook types and MCP transports agree and so record nothing.
func TestCrossCheckFromFixtures(t *testing.T) {
	t.Parallel()

	got := upstream.CrossCheck(fixtureDigest(t))

	want := []upstream.Disagreement{
		{
			Topic:      "hooks.events",
			Source:     "schemastore.plugin",
			DocsOnly:   []string{"DirectoryAdded", "MessageDisplay", "PostModelSwitch", "PreModelSwitch"},
			SourceOnly: []string{},
		},
		{
			Topic:      "hooks.events",
			Source:     "schemastore.settings",
			DocsOnly:   []string{"PostModelSwitch", "PreModelSwitch"},
			SourceOnly: []string{},
		},
		{
			Topic:      "marketplace.sources",
			Source:     "schemastore.marketplace",
			DocsOnly:   []string{"archive", "command"},
			SourceOnly: []string{},
		},
		{
			Topic:  "plugins.manifest_fields",
			Source: "schemastore.plugin",
			DocsOnly: []string{
				"defaultEnabled", "displayName",
				"experimental.evals", "experimental.monitors", "experimental.themes",
				"metadata", "workflows",
			},
			SourceOnly: []string{"monitors", "settings", "themes"},
		},
	}

	if len(got) != len(want) {
		t.Fatalf("CrossCheck() = %d disagreements, want %d: %+v", len(got), len(want), got)
	}

	for i, w := range want {
		g := got[i]
		if g.Topic != w.Topic || g.Source != w.Source {
			t.Errorf("disagreement %d = %s/%s, want %s/%s", i, g.Topic, g.Source, w.Topic, w.Source)

			continue
		}
		if !slices.Equal(g.DocsOnly, w.DocsOnly) {
			t.Errorf("%s vs %s docs_only = %v, want %v", w.Topic, w.Source, g.DocsOnly, w.DocsOnly)
		}
		if !slices.Equal(g.SourceOnly, w.SourceOnly) {
			t.Errorf("%s vs %s source_only = %v, want %v", w.Topic, w.Source, g.SourceOnly, w.SourceOnly)
		}
	}
}

// TestCrossCheckAgreementRecordsNothing covers the case the fixtures
// already exercise for hook types: equal sets produce no entry at all,
// rather than an entry with two empty arrays.
func TestCrossCheckAgreementRecordsNothing(t *testing.T) {
	t.Parallel()

	d := upstream.NewDigest()
	d.Hooks.Types = []string{"command", "http"}
	d.SchemaStore.PluginManifest.HookTypes = []string{"http", "command"}

	if got := upstream.CrossCheck(d); len(got) != 0 {
		t.Fatalf("CrossCheck() on agreeing sets = %+v, want none", got)
	}
}

// TestCrossCheckIgnoresEmptyCrossCheckSet guards the noisiest failure
// mode: a schema that carries nothing for a topic would otherwise make
// the entire documented set look like a disagreement.
func TestCrossCheckIgnoresEmptyCrossCheckSet(t *testing.T) {
	t.Parallel()

	d := upstream.NewDigest()
	d.Hooks.Events = []string{"PreToolUse", "PostToolUse"}

	if got := upstream.CrossCheck(d); len(got) != 0 {
		t.Fatalf("CrossCheck() with no schema set = %+v, want none", got)
	}
}

// TestCrossCheckReportsBothDirections checks that an item the schema has
// and the documentation does not is recorded too, not just the other way
// round.
func TestCrossCheckReportsBothDirections(t *testing.T) {
	t.Parallel()

	d := upstream.NewDigest()
	d.MCP.Transports = []string{"http", "stdio"}
	d.SchemaStore.PluginManifest.MCPTypes = []string{"http", "sse"}

	got := upstream.CrossCheck(d)
	if len(got) != 1 {
		t.Fatalf("CrossCheck() = %d disagreements, want 1: %+v", len(got), got)
	}

	if want := []string{"stdio"}; !slices.Equal(got[0].DocsOnly, want) {
		t.Errorf("docs_only = %v, want %v", got[0].DocsOnly, want)
	}
	if want := []string{"sse"}; !slices.Equal(got[0].SourceOnly, want) {
		t.Errorf("source_only = %v, want %v", got[0].SourceOnly, want)
	}
}

// TestCrossCheckSorted asserts the committed order is stable: by topic,
// then by source. Two topics compare against two different schemas, so
// insertion order alone would not be deterministic to a reader.
func TestCrossCheckSorted(t *testing.T) {
	t.Parallel()

	got := upstream.CrossCheck(fixtureDigest(t))

	keys := make([]string, 0, len(got))
	for _, g := range got {
		keys = append(keys, g.Topic+"/"+g.Source)
	}

	if !slices.IsSorted(keys) {
		t.Errorf("CrossCheck() order = %v, want sorted", keys)
	}
}

// TestExtractComputesDisagreements checks the pipeline wires the
// cross-check in, rather than leaving the digest section empty.
func TestExtractComputesDisagreements(t *testing.T) {
	t.Parallel()

	d, err := upstream.Extract(fixturePages(t), upstream.ExtractOptions{})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if len(d.Disagreements) == 0 {
		t.Fatal("Extract() left disagreements empty, want the fixture cross-check results")
	}
}
