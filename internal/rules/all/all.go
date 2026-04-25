// Package all blank-imports every rule subpackage so each rule's
// init() function runs exactly once at binary startup. The cmd
// entrypoint and any test that needs the full ruleset blank-import
// this package; rule subpackages are never imported directly.
//
// Adding a new rule means adding a new `_ "..."` line here. That is
// the deliberate friction — a new rule should not sneak into
// production without this explicit registration.
package all

import (
	// Blank imports are intentional: each rule subpackage's init()
	// registers that package's rules into the registry. Removing any
	// of these silently drops the corresponding rules from the
	// binary.
	_ "github.com/donaldgifford/claudelint/internal/rules/claudemd"
	_ "github.com/donaldgifford/claudelint/internal/rules/commands"
	_ "github.com/donaldgifford/claudelint/internal/rules/hooks"
	_ "github.com/donaldgifford/claudelint/internal/rules/marketplace"
	_ "github.com/donaldgifford/claudelint/internal/rules/mcp"
	_ "github.com/donaldgifford/claudelint/internal/rules/plugin"
	_ "github.com/donaldgifford/claudelint/internal/rules/schema"
	_ "github.com/donaldgifford/claudelint/internal/rules/security"
	_ "github.com/donaldgifford/claudelint/internal/rules/skills"
	_ "github.com/donaldgifford/claudelint/internal/rules/style"
)
