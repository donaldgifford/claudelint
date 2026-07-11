package artifact

import (
	"os"
	"path/filepath"
)

// maxPluginRootWalk bounds the ancestor walk when no root is known —
// deep enough for any real repo layout, finite if a caller passes a
// root that never matches.
const maxPluginRootWalk = 32

// MarkPluginDistributed sets a.PluginDistributed when any directory
// from the agent file's parent up to root (inclusive) contains a
// .claude-plugin/plugin.json — the layout plugin-distributed
// subagents ship in. Claude Code ignores permissionMode, mcpServers,
// and hooks on such agents.
//
// Discovery calls this after a successful parse, following the
// IndexSkillCompanions precedent: the parser stays pure over bytes
// and the wiring enriches the artifact with filesystem context. An
// empty root walks all the way up (bounded).
func MarkPluginDistributed(a *Agent, absPath, root string) {
	cleanRoot := filepath.Clean(root)
	dir := filepath.Dir(absPath)
	for range maxPluginRootWalk {
		manifest := filepath.Join(dir, ".claude-plugin", "plugin.json")
		if fi, err := os.Stat(manifest); err == nil && !fi.IsDir() {
			a.PluginDistributed = true
			return
		}
		if root != "" && filepath.Clean(dir) == cleanRoot {
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}
