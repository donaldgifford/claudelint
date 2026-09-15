package upstream

import (
	"fmt"
	"maps"
	"slices"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// This is the only file in the package that imports internal/artifact.
//
// The rendered spec page answers one question the digest alone cannot:
// of everything the documentation describes, what does claudelint
// actually read? That answer lives in internal/artifact — the key
// slices the parsers use and the canonical name sets the rules check
// against — so the renderer has to see them.
//
// The import is safe in both directions. internal/artifact imports only
// internal/diag and will never need spec data, so no cycle is possible.
// Keeping the coupling in one file means "what does upstream know about
// artifact?" is one file to read rather than a grep.
//
// The guardrail test asks the same question from package upstream_test.
// It compares; this describes. They are kept in step by
// TestCoverageMatchesTheGuardrail.

// Coverage maps a digest path to the items claudelint parses or checks
// under it. It is keyed exactly like Acknowledged, so the renderer can
// ask both the same question with the same key.
type Coverage map[string]map[string]struct{}

// DefaultCoverage is what the shipped parsers and rules read.
//
// A path absent from this map means claudelint has no opinion about
// that section: the renderer says so rather than implying a "no".
func DefaultCoverage() Coverage {
	return Coverage{
		"agents.enums.color":          setOfStrings(setKeys(artifact.AgentColors)),
		"agents.enums.effort":         setOfStrings(setKeys(artifact.AgentEffortLevels)),
		"agents.enums.isolation":      setOfStrings([]string{"worktree"}),
		"agents.enums.memory":         setOfStrings(setKeys(artifact.AgentMemoryScopes)),
		"agents.enums.model":          setOfStrings(append(setKeys(artifact.KnownModelAliases), "inherit")),
		"agents.enums.permissionMode": setOfStrings(setKeys(artifact.AgentPermissionModes)),
		"agents.frontmatter":          setOfStrings(artifact.AgentFrontmatterKeys),
		"hooks.events":                setOfStrings(setKeys(artifact.KnownHookEvents)),
		"hooks.handler_fields":        setOfStrings(artifact.HookEntryKeys),
		"hooks.types":                 setOfStrings(setKeys(artifact.KnownHookTypes)),
		"marketplace.reserved_names":  setOfStrings(setKeys(artifact.ReservedMarketplaceNames)),
		"marketplace.sources":         setOfStrings(sourceKinds()),
		"mcp.transports":              setOfStrings(setKeys(artifact.KnownTransports)),
		"plugins.manifest_fields":     setOfStrings(artifact.PluginManifestKeys),
		"skills.frontmatter":          setOfStrings(skillAndCommandKeys()),
		"tools.builtin":               setOfStrings(setKeys(artifact.KnownTools)),
	}
}

// Validate rejects a coverage entry naming a digest path that does not
// exist, which would silently render as "claudelint has no opinion".
func (c Coverage) Validate(d *Digest) error {
	tree, err := d.Tree()
	if err != nil {
		return err
	}

	for _, path := range c.Paths() {
		if _, ok := At(tree, path); !ok {
			return fmt.Errorf("coverage table: %w: %q", ErrUnknownPath, path)
		}
	}

	return nil
}

// Paths returns the covered digest paths, sorted.
func (c Coverage) Paths() []string { return slices.Sorted(maps.Keys(c)) }

// Lookup answers the renderer's question in three states rather than
// two. known is false when claudelint has no opinion about the section
// at all, which is different from knowing about it and not parsing the
// item.
func (c Coverage) Lookup(path, item string) (known, parsed bool) {
	items, ok := c[path]
	if !ok {
		return false, false
	}
	_, parsed = items[item]

	return true, parsed
}

// setKeys turns a canonical name set into a sorted slice.
func setKeys[V any](m map[string]V) []string { return slices.Sorted(maps.Keys(m)) }

// setOfStrings turns a slice into a set.
func setOfStrings(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item] = struct{}{}
	}

	return out
}

// sourceKinds is the documented marketplace source kinds as strings.
func sourceKinds() []string {
	out := make([]string, 0, len(artifact.MarketplaceSourceKinds))
	for _, k := range artifact.MarketplaceSourceKinds {
		out = append(out, string(k))
	}
	slices.Sort(out)

	return out
}

// skillAndCommandKeys is the union the digest's skills.frontmatter
// section covers: the published table lists the fields a skill or a
// command may carry, and claudelint splits them across two parsers.
func skillAndCommandKeys() []string {
	union := append(slices.Clone(artifact.SkillFrontmatterKeys), artifact.CommandFrontmatterKeys...)
	slices.Sort(union)

	return slices.Compact(union)
}
