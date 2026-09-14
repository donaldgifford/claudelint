package upstream

import (
	"maps"
	"slices"
)

// The SchemaStore schemas describe the same surface as the published
// documentation and do not agree with it. The plugin-manifest schema
// knows hook events the docs have dropped; the docs carry events no
// schema has yet. Resolving that in favour of either side would throw
// away a fact, so the two sets are compared and the difference is
// recorded under digest.disagreements.
//
// This is information, never drift: the diff partitions disagreements
// into the informational block, so a SchemaStore regeneration cannot on
// its own fail a spec-drift run (DESIGN-0006 OQ9). The documentation
// wins wherever a rule has to pick one.

// The topics compared. Each names the digest path holding the
// documented side, so a reader can go straight from a disagreement to
// the set it came from.
const (
	topicHookEvents         = "hooks.events"
	topicHookTypes          = "hooks.types"
	topicMCPTransports      = "mcp.transports"
	topicMarketplaceSources = "marketplace.sources"
	topicPluginManifest     = "plugins.manifest_fields"
)

// crossCheck is one documented set and the cross-check set it is
// compared against.
type crossCheck struct {
	// Topic is the digest path of the documented side.
	Topic string
	// Source is the id of the cross-check source.
	Source string
	// Docs is the documented set.
	Docs []string
	// Other is the cross-check set.
	Other []string
}

// CrossCheck compares every documented set against its SchemaStore
// counterpart and returns one Disagreement per pair that differs, sorted
// and free of empty entries.
//
// Hook events are compared against both schemas because both describe
// them: the plugin manifest for a plugin's bundled hooks, the settings
// schema for a user's own. They do not agree with each other either.
func CrossCheck(d *Digest) []Disagreement {
	pairs := []crossCheck{
		{
			Topic:  topicHookEvents,
			Source: SrcSchemaStorePlugin,
			Docs:   d.Hooks.Events,
			Other:  d.SchemaStore.PluginManifest.HookEvents,
		},
		{
			Topic:  topicHookEvents,
			Source: SrcSchemaStoreSettings,
			Docs:   d.Hooks.Events,
			Other:  d.SchemaStore.Settings.HookEvents,
		},
		{
			Topic:  topicHookTypes,
			Source: SrcSchemaStorePlugin,
			Docs:   d.Hooks.Types,
			Other:  d.SchemaStore.PluginManifest.HookTypes,
		},
		{
			Topic:  topicMCPTransports,
			Source: SrcSchemaStorePlugin,
			Docs:   d.MCP.Transports,
			Other:  d.SchemaStore.PluginManifest.MCPTypes,
		},
		{
			Topic:  topicMarketplaceSources,
			Source: SrcSchemaStoreMarketplace,
			Docs:   slices.Collect(maps.Keys(d.Marketplace.Sources)),
			Other:  d.SchemaStore.Marketplace.SourceKinds,
		},
		{
			Topic:  topicPluginManifest,
			Source: SrcSchemaStorePlugin,
			Docs:   fieldNames(d.Plugins.ManifestFields),
			Other:  d.SchemaStore.PluginManifest.Properties,
		},
	}

	out := make([]Disagreement, 0, len(pairs))

	for _, p := range pairs {
		// An empty cross-check set means the schema carries nothing to
		// compare against, not that the documentation invented every
		// item in its own. Reporting the whole documented set as a
		// disagreement would be noise; an extractor that genuinely
		// returned nothing is caught by its sanity floor instead.
		if len(p.Other) == 0 {
			continue
		}

		docsOnly, sourceOnly := difference(p.Docs, p.Other)
		if len(docsOnly) == 0 && len(sourceOnly) == 0 {
			continue
		}

		out = append(out, Disagreement{
			DocsOnly:   docsOnly,
			Source:     p.Source,
			SourceOnly: sourceOnly,
			Topic:      p.Topic,
		})
	}

	return normalizeDisagreements(out)
}

// difference returns the members of a missing from b and the members of
// b missing from a.
func difference(a, b []string) (onlyA, onlyB []string) {
	inA := setOf(a)
	inB := setOf(b)

	for _, s := range a {
		if _, ok := inB[s]; !ok {
			onlyA = append(onlyA, s)
		}
	}

	for _, s := range b {
		if _, ok := inA[s]; !ok {
			onlyB = append(onlyB, s)
		}
	}

	return onlyA, onlyB
}

// setOf indexes a slice for membership tests.
func setOf(in []string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for _, s := range in {
		out[s] = struct{}{}
	}

	return out
}

// fieldNames is the distinct names of a row set. The plugin manifest
// tables carry a name under more than one group; the schema's property
// list does not have groups, so the comparison is by name alone.
func fieldNames(fields []Field) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, f.Name)
	}

	return out
}
