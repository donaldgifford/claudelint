package marketplace

import (
	"fmt"
	"sort"
	"strings"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&renamesValid{}) }

// renamesValid errors when a renames{} migration entry cannot
// terminate: every chain must end at null (plugin removed) or at a
// name listed in plugins[], and no chain may cycle. A dangling or
// cyclic rename strands existing installs mid-migration — Claude Code
// follows the chain to find the current plugin and never arrives.
//
// The parser stores no per-entry ranges for renames{}, so diagnostics
// anchor at the manifest's name field rather than (0,0).
type renamesValid struct{}

func (*renamesValid) ID() string                     { return "marketplace/renames-valid" }
func (*renamesValid) Category() string               { return categorySchema }
func (*renamesValid) DefaultSeverity() diag.Severity { return diag.SeverityError }
func (*renamesValid) DefaultOptions() map[string]any { return nil }
func (*renamesValid) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindMarketplace}
}

func (*renamesValid) HelpURI() string { return rules.DefaultHelpURI("marketplace/renames-valid") }

func (r *renamesValid) Check(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
	m, ok := a.(*artifact.Marketplace)
	if !ok || len(m.Renames) == 0 {
		return nil
	}
	listed := make(map[string]struct{}, len(m.Plugins))
	for i := range m.Plugins {
		listed[m.Plugins[i].Name] = struct{}{}
	}

	keys := make([]string, 0, len(m.Renames))
	for k := range m.Renames {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out []diag.Diagnostic
	emit := func(msg string) {
		out = append(out, diag.Diagnostic{
			RuleID:  r.ID(),
			Path:    m.Path(),
			Range:   m.NameRange,
			Message: msg,
		})
	}

	// Per-link check: each target must be null (""), a listed plugin,
	// or another rename key. One diagnostic per broken link, not per
	// chain that crosses it.
	for _, k := range keys {
		t := m.Renames[k]
		if t == "" {
			continue // removed — valid terminal
		}
		if _, ok := listed[t]; ok {
			continue
		}
		if _, ok := m.Renames[t]; ok {
			continue
		}
		emit(fmt.Sprintf(
			"renames target %q (from %q) is neither null nor a listed plugin", t, k))
	}

	// Cycle check: emit once per cycle, for its smallest member.
	reported := make(map[string]struct{})
	for _, k := range keys {
		if _, done := reported[k]; done {
			continue
		}
		if cycle := findRenameCycle(m.Renames, k); len(cycle) > 0 {
			for _, member := range cycle {
				reported[member] = struct{}{}
			}
			emit(fmt.Sprintf(
				"renames chain %s never terminates (cycle)",
				strings.Join(append(cycle, cycle[0]), " -> ")))
		}
	}
	return out
}

// findRenameCycle walks the chain from start and returns the cycle's
// members (rotated to begin at the smallest name) when the walk
// revisits a node, or nil when the chain terminates.
func findRenameCycle(renames map[string]string, start string) []string {
	seen := map[string]int{}
	var path []string
	cur := start
	for {
		if at, ok := seen[cur]; ok {
			cycle := path[at:]
			rotateToSmallest(cycle)
			return cycle
		}
		seen[cur] = len(path)
		path = append(path, cur)
		next, ok := renames[cur]
		if !ok || next == "" {
			return nil
		}
		cur = next
	}
}

// rotateToSmallest rotates cycle in place so it begins at its
// lexicographically smallest member, giving deterministic output no
// matter which member the walk entered through.
func rotateToSmallest(cycle []string) {
	if len(cycle) < 2 {
		return
	}
	smallest := 0
	for i, v := range cycle {
		if v < cycle[smallest] {
			smallest = i
		}
	}
	rotated := append(append([]string{}, cycle[smallest:]...), cycle[:smallest]...)
	copy(cycle, rotated)
}
