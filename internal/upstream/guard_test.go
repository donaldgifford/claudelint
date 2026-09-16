package upstream_test

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/rules/hooks"
	"github.com/donaldgifford/claudelint/internal/upstream"
)

// The guardrail ties the committed digest to the constants the rules
// actually use, so upstream drift becomes a failing test that names the
// Go identifier to edit rather than a weekly issue somebody has to read.
// It is the upstream analogue of TestRulesetFingerprint, and like that
// test it runs offline: the digest and the acknowledgements are
// compiled in.
//
// This file lives in the external test package on purpose. The library
// imports nothing else from this module, and the guardrail has to reach
// into internal/artifact and three rule packages; keeping it out of
// package upstream is what lets both be true.

// TestGuardrail is one subtest per row of DESIGN-0006 §7.
//
// The subtests run in sequence, not in parallel: each records which
// acknowledgements it used, and the last one fails any that went
// unused. A shared accumulator and t.Parallel do not mix.
func TestGuardrail(t *testing.T) {
	digest, ack, err := upstream.LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error = %v", err)
	}

	g := &guard{digest: digest, ack: ack}

	t.Run("hooks.events", g.hookEvents)
	t.Run("hooks.types", g.hookTypes)
	t.Run("hooks.timeout_defaults", g.hookTimeouts)
	t.Run("tools.builtin", g.tools)
	t.Run("agents.enums", g.agentEnums)
	t.Run("frontmatter keys", g.frontmatterKeys)
	t.Run("marketplace.sources", g.marketplaceSources)
	t.Run("marketplace.reserved_names", g.reservedNames)
	t.Run("mcp.transports", g.mcpTransports)
	t.Run("acknowledgements are live", g.acknowledgementsAreLive)
}

// guard carries the loaded digest and acknowledgements between
// subtests, and records which acknowledgements were used so a stale one
// can be reported.
type guard struct {
	digest *upstream.Digest
	ack    upstream.Acknowledged

	// used records path -> item for every acknowledgement that excused a
	// real difference.
	used map[string]map[string]bool
}

func (g *guard) markUsed(path, item string) {
	if g.used == nil {
		g.used = make(map[string]map[string]bool)
	}
	if g.used[path] == nil {
		g.used[path] = make(map[string]bool)
	}
	g.used[path][item] = true
}

// equalSets is the relation most rows use: the documented set and the
// encoded set must match, item for item, unless an item is
// acknowledged.
func (g *guard) equalSets(t *testing.T, path, identifier string, documented, encoded []string) {
	t.Helper()

	if msg := g.equalSetsDrift(path, identifier, documented, encoded); msg != "" {
		t.Error(msg)
	}
}

// equalSetsDrift is the check itself, returning the failure message or
// "". Keeping it separate from the assertion is what lets the negative
// tests below drive it with synthetic data.
func (g *guard) equalSetsDrift(path, identifier string, documented, encoded []string) string {
	missing := g.unacknowledged(path, difference(documented, encoded))
	extra := g.unacknowledged(path, difference(encoded, documented))

	if len(missing) == 0 && len(extra) == 0 {
		return ""
	}

	return driftMessage(path, identifier, missing, extra)
}

// covers is the one-way relation the key lists use: every documented
// field has to be parsed or acknowledged. The reverse is allowed, since
// the parsers read a few keys the published tables do not list.
func (g *guard) covers(t *testing.T, path, identifier string, documented, encoded []string) {
	t.Helper()

	if msg := g.coversDrift(path, identifier, documented, encoded); msg != "" {
		t.Error(msg)
	}
}

func (g *guard) coversDrift(path, identifier string, documented, encoded []string) string {
	missing := g.unacknowledged(path, difference(documented, encoded))
	if len(missing) == 0 {
		return ""
	}

	return driftMessage(path, identifier, missing, nil)
}

// staleAcknowledgements returns the acknowledgements that no longer
// excuse a real difference.
func (g *guard) staleAcknowledgements() []string {
	stale := make([]string, 0, len(g.ack))

	for _, path := range g.ack.Paths() {
		for _, item := range g.ack.Items(path) {
			if !g.used[path][item] {
				stale = append(stale, path+" / "+item)
			}
		}
	}

	return stale
}

// unacknowledged filters out the items excused in acknowledged.json and
// records each one it used.
func (g *guard) unacknowledged(path string, items []string) []string {
	var out []string

	for _, item := range items {
		if g.ack.Has(path, item) {
			g.markUsed(path, item)

			continue
		}
		out = append(out, item)
	}

	return out
}

// driftMessage follows the fingerprint test's shape: the identifier,
// the digest path, the delta, and the two ways to resolve it.
func driftMessage(path, identifier string, missing, extra []string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s is out of date with the documented %s.\n\n", identifier, path)

	if len(missing) > 0 {
		fmt.Fprintf(&b, "  documented upstream, missing from %s:\n    %s\n",
			identifier, strings.Join(missing, ", "))
	}
	if len(extra) > 0 {
		fmt.Fprintf(&b, "  in %s, not documented upstream:\n    %s\n",
			identifier, strings.Join(extra, ", "))
	}

	fmt.Fprintf(&b, `
Resolve it either way:
  - change %s to match, or
  - record why it differs in internal/upstream/acknowledged.json under
    %q, with a reason naming the rule or phase that would resolve it.

If upstream itself moved, run "just spec-sync" first so the digest
reflects today's documentation.`, identifier, path)

	return b.String()
}

func (g *guard) hookEvents(t *testing.T) {
	g.equalSets(t, "hooks.events", "artifact.KnownHookEvents",
		g.digest.Hooks.Events, setKeys(artifact.KnownHookEvents))
}

func (g *guard) hookTypes(t *testing.T) {
	g.equalSets(t, "hooks.types", "artifact.KnownHookTypes",
		g.digest.Hooks.Types, setKeys(artifact.KnownHookTypes))
}

// hookTimeouts compares the per-type defaults the timeout-present rule
// cites in its message. A wrong number there is worse than a missing
// rule: it tells the reader something false about the runtime.
func (g *guard) hookTimeouts(t *testing.T) {
	const path = "hooks.timeout_defaults.by_type"

	for _, typ := range slices.Sorted(maps.Keys(g.digest.Hooks.TimeoutDefaults.ByType)) {
		documented := g.digest.Hooks.TimeoutDefaults.ByType[typ]
		if got := hooks.DefaultTimeoutSecs(typ); got != documented {
			t.Errorf(
				"hooks.DefaultTimeoutSecs(%q) = %d, documented %d (%s).\n"+
					"Update the rule's defaults, or acknowledge the difference under %q.",
				typ, got, documented, path, path)
		}
	}

	// Per-event overrides are a separate surface. No rule reads them
	// yet, so each documented override is acknowledged rather than
	// compared.
	byEvent := "hooks.timeout_defaults.by_event"
	for _, event := range slices.Sorted(maps.Keys(g.digest.Hooks.TimeoutDefaults.ByEvent)) {
		if !g.ack.Has(byEvent, event) {
			t.Errorf(
				"%s documents an override for %q that no rule reads.\n"+
					"Consume it in a rule, or acknowledge it under %q.",
				byEvent, event, byEvent)

			continue
		}
		g.markUsed(byEvent, event)
	}
}

func (g *guard) tools(t *testing.T) {
	g.equalSets(t, "tools.builtin", "artifact.KnownTools",
		g.digest.Tools.Builtin, setKeys(artifact.KnownTools))

	g.deprecatedTools(t)
}

// deprecatedTools checks the table the tools-known rules consult before
// reporting an unknown tool. A removed or renamed tool must be gone
// from the documented list, and a merely deprecated one must still be
// in it: either way round, a mismatch means the table is stale and the
// rule would give the wrong advice.
func (g *guard) deprecatedTools(t *testing.T) {
	t.Helper()

	documented := setOf(g.digest.Tools.Builtin)

	for _, name := range slices.Sorted(maps.Keys(artifact.DeprecatedTools)) {
		entry := artifact.DeprecatedTools[name]
		_, isDocumented := documented[name]

		switch entry.Status {
		case artifact.ToolRemoved, artifact.ToolRenamed:
			if isDocumented {
				t.Errorf(
					"artifact.DeprecatedTools[%q] says %s, but the tools reference still lists it.\n"+
						"Remove the entry, or correct its status.",
					name, entry.Status)
			}
		case artifact.ToolDeprecated:
			if !isDocumented {
				t.Errorf(
					"artifact.DeprecatedTools[%q] says deprecated, but the tools reference no longer lists it.\n"+
						"Change the status to removed, and say what replaced it.",
					name)
			}
		default:
			t.Errorf("artifact.DeprecatedTools[%q] has an unknown status %q", name, entry.Status)
		}
	}
}

// agentEnums covers the five enum rows in one loop. The model row is
// special: the documented cell lists the aliases plus "inherit", which
// the rule accepts through IsValidModelRef rather than the alias map.
func (g *guard) agentEnums(t *testing.T) {
	tests := []struct {
		field      string
		identifier string
		encoded    []string
	}{
		{"color", "artifact.AgentColors", setKeys(artifact.AgentColors)},
		{"effort", "artifact.AgentEffortLevels", setKeys(artifact.AgentEffortLevels)},
		{"memory", "artifact.AgentMemoryScopes", setKeys(artifact.AgentMemoryScopes)},
		{"model", "artifact.KnownModelAliases", append(setKeys(artifact.KnownModelAliases), "inherit")},
		{"permissionMode", "artifact.AgentPermissionModes", setKeys(artifact.AgentPermissionModes)},
	}

	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			documented, ok := g.digest.Agents.Enums[tc.field]
			if !ok {
				t.Fatalf("the digest has no agents.enums.%s; did the docs table change?", tc.field)
			}
			g.equalSets(t, "agents.enums."+tc.field, tc.identifier, documented, tc.encoded)
		})
	}

	t.Run("isolation", func(t *testing.T) {
		// isolation has one documented value and is checked directly by
		// agents/field-enums rather than through a set.
		g.covers(t, "agents.enums.isolation", "the agents/field-enums isolation check",
			g.digest.Agents.Enums["isolation"], []string{"worktree"})
	})
}

func (g *guard) frontmatterKeys(t *testing.T) {
	// Skills and commands share one documented table; a key parsed by
	// either satisfies it.
	skillAndCommand := append(
		slices.Clone(artifact.SkillFrontmatterKeys),
		artifact.CommandFrontmatterKeys...,
	)

	tests := []struct {
		path       string
		identifier string
		documented []string
		encoded    []string
	}{
		{
			path:       "skills.frontmatter",
			identifier: "artifact.SkillFrontmatterKeys + CommandFrontmatterKeys",
			documented: fieldNames(g.digest.Skills.Frontmatter),
			encoded:    skillAndCommand,
		},
		{
			path:       "agents.frontmatter",
			identifier: "artifact.AgentFrontmatterKeys",
			documented: fieldNames(g.digest.Agents.Frontmatter),
			encoded:    artifact.AgentFrontmatterKeys,
		},
		{
			path:       "plugins.manifest_fields",
			identifier: "artifact.PluginManifestKeys",
			documented: fieldNames(g.digest.Plugins.ManifestFields),
			encoded:    artifact.PluginManifestKeys,
		},
		{
			path:       "hooks.handler_fields",
			identifier: "artifact.HookEntryKeys",
			documented: handlerFieldNames(g.digest.Hooks.HandlerFields),
			encoded:    artifact.HookEntryKeys,
		},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			g.covers(t, tc.path, tc.identifier, tc.documented, tc.encoded)
		})
	}
}

// marketplaceSources checks that every documented plugin source kind
// has a MarketplaceSourceKind the parser can produce. A kind with no
// constant is a source shape claudelint classifies as invalid, which
// turns a valid manifest into an error.
func (g *guard) marketplaceSources(t *testing.T) {
	encoded := make([]string, 0, len(artifact.MarketplaceSourceKinds))
	for _, k := range artifact.MarketplaceSourceKinds {
		encoded = append(encoded, string(k))
	}

	g.equalSets(t, "marketplace.sources", "artifact.MarketplaceSourceKinds",
		slices.Sorted(maps.Keys(g.digest.Marketplace.Sources)), encoded)
}

func (g *guard) reservedNames(t *testing.T) {
	g.equalSets(t, "marketplace.reserved_names",
		"artifact.ReservedMarketplaceNames",
		g.digest.Marketplace.ReservedNames,
		setKeys(artifact.ReservedMarketplaceNames))
}

func (g *guard) mcpTransports(t *testing.T) {
	g.equalSets(t, "mcp.transports", "artifact.KnownTransports",
		g.digest.MCP.Transports, setKeys(artifact.KnownTransports))
}

// acknowledgementsAreLive is what stops the file becoming a graveyard:
// an acknowledgement that no longer excuses a real difference fails,
// so removing it is forced rather than remembered.
func (g *guard) acknowledgementsAreLive(t *testing.T) {
	for _, stale := range g.staleAcknowledgements() {
		t.Errorf(
			"acknowledged.json excuses %s, but that is no longer a difference.\n"+
				"Delete the entry: the code and the documentation now agree.",
			stale)
	}
}

// setKeys returns the keys of a set, sorted.
func setKeys[V any](m map[string]V) []string { return slices.Sorted(maps.Keys(m)) }

// setOf indexes a slice for membership tests.
func setOf(in []string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for _, s := range in {
		out[s] = struct{}{}
	}

	return out
}

// difference returns the members of a that are not in b, sorted.
func difference(a, b []string) []string {
	inB := setOf(b)

	out := make([]string, 0, len(a))
	for _, s := range a {
		if _, ok := inB[s]; !ok {
			out = append(out, s)
		}
	}
	slices.Sort(out)

	return slices.Compact(out)
}

// handlerFieldNames flattens the per-type handler tables into one set of
// field names.
func handlerFieldNames(byType map[string][]upstream.Field) []string {
	out := make([]string, 0, len(byType))
	for _, fields := range byType {
		out = append(out, fieldNames(fields)...)
	}
	slices.Sort(out)

	return slices.Compact(out)
}
