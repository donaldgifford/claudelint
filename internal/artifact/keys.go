package artifact

import "slices"

// The keys the parsers read, named once.
//
// Each parser looks its keys up through these constants, and the
// exported per-kind slices below are built from the same names. The
// upstream guardrail test in internal/upstream compares those slices
// against the documented field tables in the committed spec digest, so
// a field Anthropic adds to a frontmatter table becomes a failing test
// naming the slice to extend, rather than a key the parser silently
// ignores.
//
// The constants are unexported because nothing outside the package
// needs one; the slices are the contract.
const (
	keyAgent                  = "agent"
	keyAgents                 = "agents"
	keyAllowedTools           = "allowed-tools"
	keyArgumentHint           = "argument-hint"
	keyArgs                   = "args"
	keyAsync                  = "async"
	keyBackground             = "background"
	keyColor                  = "color"
	keyCommand                = "command"
	keyCommands               = "commands"
	keyContext                = "context"
	keyDescription            = "description"
	keyDisableModelInvocation = "disable-model-invocation"
	keyDisallowedTools        = "disallowed-tools"
	keyDisallowedToolsCamel   = "disallowedTools"
	keyEffort                 = "effort"
	keyHooks                  = "hooks"
	keyInitialPrompt          = "initialPrompt"
	keyIsolation              = "isolation"
	keyMaxTurns               = "maxTurns"
	keyMCPServers             = "mcpServers"
	keyMemory                 = "memory"
	keyModel                  = "model"
	keyName                   = "name"
	keyPermissionMode         = "permissionMode"
	keyPrompt                 = "prompt"
	keyServer                 = "server"
	keyShell                  = "shell"
	keySkills                 = "skills"
	keyTimeout                = "timeout"
	keyTool                   = "tool"
	keyTools                  = "tools"
	keyType                   = "type"
	keyURL                    = "url"
	keyUserInvocable          = "user-invocable"
	keyVersion                = "version"
	keyWhenToUse              = "when_to_use"
)

// SkillFrontmatterKeys is every SKILL.md frontmatter key ParseSkill
// reads, sorted.
var SkillFrontmatterKeys = sortedKeys(
	keyAgent,
	keyAllowedTools,
	keyContext,
	keyDescription,
	keyDisableModelInvocation,
	keyDisallowedTools,
	keyModel,
	keyName,
	keyUserInvocable,
	keyWhenToUse,
)

// CommandFrontmatterKeys is every slash-command frontmatter key
// ParseCommand reads, sorted. It overlaps SkillFrontmatterKeys almost
// entirely: a command has an argument-hint where a skill has a name.
var CommandFrontmatterKeys = sortedKeys(
	keyAgent,
	keyAllowedTools,
	keyArgumentHint,
	keyContext,
	keyDescription,
	keyDisableModelInvocation,
	keyDisallowedTools,
	keyModel,
	keyUserInvocable,
	keyWhenToUse,
)

// AgentFrontmatterKeys is every subagent frontmatter key ParseAgent
// reads, sorted.
var AgentFrontmatterKeys = sortedKeys(
	keyBackground,
	keyColor,
	keyDescription,
	keyDisallowedToolsCamel,
	keyEffort,
	keyHooks,
	keyInitialPrompt,
	keyIsolation,
	keyMaxTurns,
	keyMCPServers,
	keyMemory,
	keyModel,
	keyName,
	keyPermissionMode,
	keySkills,
	keyTools,
)

// PluginManifestKeys is every plugin.json key ParsePlugin reads,
// sorted.
var PluginManifestKeys = sortedKeys(
	keyAgents,
	keyCommands,
	keyDescription,
	keyName,
	keySkills,
	keyVersion,
)

// HookEntryKeys is every key of one hook handler object that ParseHook
// reads, sorted. The enclosing matcher group's own "matcher" key is not
// one of them: it describes the group, not the handler.
var HookEntryKeys = sortedKeys(
	keyArgs,
	keyAsync,
	keyCommand,
	keyPrompt,
	keyServer,
	keyShell,
	keyTimeout,
	keyTool,
	keyType,
	keyURL,
)

// sortedKeys returns the given keys sorted and deduplicated, so every
// exported list is in a stable order whatever order it was written in.
func sortedKeys(keys ...string) []string {
	out := slices.Clone(keys)
	slices.Sort(out)

	return slices.Compact(out)
}
