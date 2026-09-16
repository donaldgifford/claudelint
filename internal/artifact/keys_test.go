package artifact_test

import (
	"slices"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// TestKeyListsAreSortedAndUnique is what lets the guardrail test
// compare them against the digest with a plain set difference.
func TestKeyListsAreSortedAndUnique(t *testing.T) {
	t.Parallel()

	lists := map[string][]string{
		"AgentFrontmatterKeys":   artifact.AgentFrontmatterKeys,
		"CommandFrontmatterKeys": artifact.CommandFrontmatterKeys,
		"HookEntryKeys":          artifact.HookEntryKeys,
		"PluginManifestKeys":     artifact.PluginManifestKeys,
		"SkillFrontmatterKeys":   artifact.SkillFrontmatterKeys,
	}

	for name, keys := range lists {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if len(keys) == 0 {
				t.Fatal("key list is empty")
			}
			if !slices.IsSorted(keys) {
				t.Errorf("%s is not sorted: %v", name, keys)
			}
			if len(slices.Compact(slices.Clone(keys))) != len(keys) {
				t.Errorf("%s has duplicates: %v", name, keys)
			}
			for _, k := range keys {
				if k == "" {
					t.Errorf("%s contains an empty key", name)
				}
			}
		})
	}
}

// TestKeyListsMatchTheParsers pins the lists to what the parsers
// actually read. A key added to a parser without being added here would
// make the upstream guardrail pass on a field nothing consumes.
func TestKeyListsMatchTheParsers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  []string
		want []string
	}{
		{
			name: "skill",
			got:  artifact.SkillFrontmatterKeys,
			want: []string{
				"agent", "allowed-tools", "context", "description",
				"disable-model-invocation", "disallowed-tools", "model", "name",
				"user-invocable", "when_to_use",
			},
		},
		{
			name: "command",
			got:  artifact.CommandFrontmatterKeys,
			want: []string{
				"agent", "allowed-tools", "argument-hint", "context", "description",
				"disable-model-invocation", "disallowed-tools", "model",
				"user-invocable", "when_to_use",
			},
		},
		{
			name: "agent",
			got:  artifact.AgentFrontmatterKeys,
			want: []string{
				"background", "color", "description", "disallowedTools", "effort",
				"hooks", "initialPrompt", "isolation", "maxTurns", "mcpServers",
				"memory", "model", "name", "permissionMode", "skills", "tools",
			},
		},
		{
			name: "plugin",
			got:  artifact.PluginManifestKeys,
			want: []string{"agents", "commands", "description", "name", "skills", "version"},
		},
		{
			name: "hook entry",
			got:  artifact.HookEntryKeys,
			want: []string{
				"args", "async", "command", "prompt", "server", "shell", "timeout",
				"tool", "type", "url",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if !slices.Equal(tc.got, tc.want) {
				t.Errorf("keys = %v, want %v", tc.got, tc.want)
			}
		})
	}
}

// TestParsersStillReadEveryKey is the behavioural half: a document
// setting every listed key round-trips through the parser with those
// values visible, so routing the parsers through the constants changed
// nothing.
func TestParsersStillReadEveryKey(t *testing.T) {
	t.Parallel()

	src := []byte(`---
name: demo
description: a demo skill
model: sonnet
when_to_use: always
context: extra
agent: reviewer
allowed-tools: [Read]
disallowed-tools: [Bash]
disable-model-invocation: true
user-invocable: false
---

body
`)

	skill, perr := artifact.ParseSkill("SKILL.md", src)
	if perr != nil {
		t.Fatalf("ParseSkill() error = %v", perr)
	}

	for _, key := range artifact.SkillFrontmatterKeys {
		if _, ok := skill.Frontmatter.Keys[key]; !ok {
			t.Errorf("frontmatter is missing %q, so the fixture no longer covers it", key)
		}
	}

	if skill.Name != "demo" || skill.Model != "sonnet" || skill.Agent != "reviewer" {
		t.Errorf("ParseSkill() = %+v, want the fixture values", skill)
	}
	if !skill.DisableModelInvocation {
		t.Error("disable-model-invocation was not read")
	}
	if skill.UserInvocable == nil || *skill.UserInvocable {
		t.Error("user-invocable was not read as false")
	}
	if len(skill.AllowedTools) != 1 || len(skill.DisallowedTools) != 1 {
		t.Errorf("tool lists = %v / %v, want one each", skill.AllowedTools, skill.DisallowedTools)
	}
}
