# .claudelint.hcl — configuration for claudelint.
#
# See https://github.com/donaldgifford/claudelint for the full rule
# reference. This file is a working default; delete blocks you don't
# need and use `claudelint rules <id>` to inspect per-rule options.

claudelint {
  version = "1"
}

# Per-ArtifactKind defaults. The label must be one of "claude_md",
# "skill", "command", "agent", "hook", or "plugin".
#
# rules "skill" {
#   default_severity = "warning"
# }

# Per-rule overrides. Label is the rule ID; see `claudelint rules`.
#
# rule "skills/body-size" {
#   severity = "warning"
#   options  = {
#     max_words = 1500
#   }
# }

# Global path-glob exclusions (applied to every rule).
#
# ignore {
#   paths = [
#     "vendor/**",
#     "node_modules/**",
#   ]
# }

# Output format: "text" (default), "json", or "github".
#
# output {
#   format = "text"
# }
