package artifact

import (
	"bytes"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/diag"
)

func TestParseMarkdownNoFrontmatter(t *testing.T) {
	src := []byte("# Hello\n\njust markdown.\n")
	doc, err := parseMarkdown("CLAUDE.md", src)
	if err != nil {
		t.Fatalf("parseMarkdown = %v, want nil", err)
	}
	if !doc.Frontmatter.Block.IsZero() {
		t.Errorf("no-frontmatter doc should have zero Block range, got %+v", doc.Frontmatter.Block)
	}
	if doc.Body.Start.Offset != 0 || doc.Body.End.Offset != len(src) {
		t.Errorf("Body range = %+v, want 0..%d", doc.Body, len(src))
	}
}

func TestParseMarkdownFrontmatterKeysAndRanges(t *testing.T) {
	src := []byte("---\nname: writer\ndescription: write things\n---\n# body\n")

	doc, err := parseMarkdown("skills/writer/SKILL.md", src)
	if err != nil {
		t.Fatalf("parseMarkdown = %v, want nil", err)
	}
	if _, ok := doc.Frontmatter.Keys["name"]; !ok {
		t.Errorf("Keys missing 'name'; got %v", keysOf(doc.Frontmatter.Keys))
	}
	if _, ok := doc.Frontmatter.Keys["description"]; !ok {
		t.Errorf("Keys missing 'description'; got %v", keysOf(doc.Frontmatter.Keys))
	}

	// 'name' key starts on file line 2, column 1.
	nameRange := doc.Frontmatter.Keys["name"]
	if nameRange.Start.Line != 2 || nameRange.Start.Column != 1 {
		t.Errorf("name range start = %+v, want line 2 col 1", nameRange.Start)
	}
	// 'description' key starts on file line 3, column 1.
	descRange := doc.Frontmatter.Keys["description"]
	if descRange.Start.Line != 3 || descRange.Start.Column != 1 {
		t.Errorf("description range start = %+v, want line 3 col 1", descRange.Start)
	}
}

func TestParseSkillExtractsTypedFields(t *testing.T) {
	src := []byte(
		"---\n" +
			"name: writer\n" +
			"description: write things\n" +
			"model: sonnet\n" +
			"allowed-tools:\n" +
			"  - Read\n" +
			"  - Write\n" +
			"---\n# body\n")

	s, err := ParseSkill("skills/writer/SKILL.md", src)
	if err != nil {
		t.Fatalf("ParseSkill = %v, want nil", err)
	}
	if s.Name != "writer" {
		t.Errorf("Name = %q, want writer", s.Name)
	}
	if s.Description != "write things" {
		t.Errorf("Description = %q, want \"write things\"", s.Description)
	}
	if s.Model != "sonnet" {
		t.Errorf("Model = %q, want sonnet", s.Model)
	}
	if len(s.AllowedTools) != 2 || s.AllowedTools[0] != "Read" || s.AllowedTools[1] != "Write" {
		t.Errorf("AllowedTools = %v, want [Read Write]", s.AllowedTools)
	}
	if s.Kind() != KindSkill {
		t.Errorf("Kind = %q, want %q", s.Kind(), KindSkill)
	}
}

func TestParseCommandAndAgent(t *testing.T) {
	cmdSrc := []byte(
		"---\ndescription: review\nargument-hint: <pr>\nallowed-tools: Read\n---\n")
	c, perr := ParseCommand(".claude/commands/review.md", cmdSrc)
	if perr != nil {
		t.Fatalf("ParseCommand = %v", perr)
	}
	if c.Description != "review" || c.ArgumentHint != "<pr>" || len(c.AllowedTools) != 1 {
		t.Errorf("Command = %+v", c)
	}

	agSrc := []byte("---\nname: scribe\ndescription: notes\ntools: [Read]\n---\n")
	a, perr := ParseAgent(".claude/agents/scribe.md", agSrc)
	if perr != nil {
		t.Fatalf("ParseAgent = %v", perr)
	}
	if a.Name != "scribe" || a.Description != "notes" || len(a.Tools) != 1 {
		t.Errorf("Agent = %+v", a)
	}
}

// TestParseSkillMergedModelFields covers the frontmatter fields the
// merged skill/command model added: when_to_use, context/agent fork
// pairing, invocability toggles, and disallowed-tools.
func TestParseSkillMergedModelFields(t *testing.T) {
	src := []byte(
		"---\n" +
			"name: deployer\n" +
			"description: deploy things\n" +
			"when_to_use: when the user asks to ship\n" +
			"context: fork\n" +
			"agent: shipper\n" +
			"disable-model-invocation: true\n" +
			"user-invocable: false\n" +
			"disallowed-tools: Write, Edit\n" +
			"---\nbody\n")
	s, perr := ParseSkill("skills/deployer/SKILL.md", src)
	if perr != nil {
		t.Fatalf("ParseSkill = %v", perr)
	}
	if s.WhenToUse != "when the user asks to ship" {
		t.Errorf("WhenToUse = %q", s.WhenToUse)
	}
	if s.Context != "fork" || s.Agent != "shipper" {
		t.Errorf("Context/Agent = %q/%q, want fork/shipper", s.Context, s.Agent)
	}
	if !s.DisableModelInvocation {
		t.Errorf("DisableModelInvocation = false, want true")
	}
	if s.UserInvocable == nil || *s.UserInvocable {
		t.Errorf("UserInvocable = %v, want declared false", s.UserInvocable)
	}
	if len(s.DisallowedTools) != 2 || s.DisallowedTools[0] != "Write" || s.DisallowedTools[1] != "Edit" {
		t.Errorf("DisallowedTools = %v, want [Write Edit]", s.DisallowedTools)
	}
}

// TestParseSkillDefaultsWhenFieldsAbsent pins the absent-key behavior:
// user-invocable stays nil (runtime default true) rather than false.
func TestParseSkillDefaultsWhenFieldsAbsent(t *testing.T) {
	src := []byte("---\nname: minimal\ndescription: d\n---\nbody\n")
	s, perr := ParseSkill("skills/minimal/SKILL.md", src)
	if perr != nil {
		t.Fatalf("ParseSkill = %v", perr)
	}
	if s.UserInvocable != nil {
		t.Errorf("UserInvocable = %v, want nil for absent key", *s.UserInvocable)
	}
	if s.DisableModelInvocation {
		t.Errorf("DisableModelInvocation = true, want false for absent key")
	}
	if s.WhenToUse != "" || s.Context != "" || s.Agent != "" || s.DisallowedTools != nil {
		t.Errorf("absent fields should be zero: %+v", s)
	}
}

// TestParseCommandMergedModelFields mirrors the skill test for the
// command parser, including the command-level model override.
func TestParseCommandMergedModelFields(t *testing.T) {
	src := []byte(
		"---\n" +
			"description: run the deploy\n" +
			"model: haiku\n" +
			"when_to_use: shipping time\n" +
			"context: fork\n" +
			"agent: shipper\n" +
			"user-invocable: true\n" +
			"disallowed-tools:\n  - Write\n" +
			"---\nbody\n")
	c, perr := ParseCommand(".claude/commands/deploy.md", src)
	if perr != nil {
		t.Fatalf("ParseCommand = %v", perr)
	}
	if c.Model != "haiku" {
		t.Errorf("Model = %q, want haiku", c.Model)
	}
	if c.WhenToUse != "shipping time" || c.Context != "fork" || c.Agent != "shipper" {
		t.Errorf("WhenToUse/Context/Agent = %q/%q/%q", c.WhenToUse, c.Context, c.Agent)
	}
	if c.UserInvocable == nil || !*c.UserInvocable {
		t.Errorf("UserInvocable = %v, want declared true", c.UserInvocable)
	}
	if c.DisableModelInvocation {
		t.Errorf("DisableModelInvocation = true, want false for absent key")
	}
	if len(c.DisallowedTools) != 1 || c.DisallowedTools[0] != "Write" {
		t.Errorf("DisallowedTools = %v, want [Write]", c.DisallowedTools)
	}
}

// TestParseAgentFullFieldSet covers the documented 16-field subagent
// frontmatter spec (DESIGN-0005 §1).
func TestParseAgentFullFieldSet(t *testing.T) {
	src := []byte(`---
name: reviewer
description: reviews changes
tools: Read, Grep, Bash(git diff:*)
disallowedTools: Write, Edit
model: opus
permissionMode: acceptEdits
maxTurns: 12
skills:
  - go
  - docz
mcpServers:
  - github
hooks:
  PreToolUse:
    - matcher: Bash
      hooks:
        - type: command
          command: ./guard.sh
memory: project
background: true
effort: high
isolation: worktree
color: cyan
initialPrompt: Review the latest diff.
---
body
`)
	a, perr := ParseAgent(".claude/agents/reviewer.md", src)
	if perr != nil {
		t.Fatalf("ParseAgent = %v", perr)
	}
	if len(a.Tools) != 3 || a.Tools[2] != "Bash(git diff:*)" {
		t.Errorf("Tools = %v", a.Tools)
	}
	if len(a.DisallowedTools) != 2 || a.DisallowedTools[0] != "Write" {
		t.Errorf("DisallowedTools = %v", a.DisallowedTools)
	}
	if a.Model != "opus" || a.PermissionMode != "acceptEdits" {
		t.Errorf("Model/PermissionMode = %q/%q", a.Model, a.PermissionMode)
	}
	if a.MaxTurns != 12 {
		t.Errorf("MaxTurns = %d, want 12", a.MaxTurns)
	}
	if len(a.Skills) != 2 || a.Skills[0] != "go" {
		t.Errorf("Skills = %v", a.Skills)
	}
	if !a.HasMCPServers || !a.HasHooks {
		t.Errorf("HasMCPServers/HasHooks = %v/%v, want true/true", a.HasMCPServers, a.HasHooks)
	}
	if a.Memory != "project" || !a.Background {
		t.Errorf("Memory/Background = %q/%v", a.Memory, a.Background)
	}
	if a.Effort != "high" || a.Isolation != "worktree" || a.Color != "cyan" {
		t.Errorf("Effort/Isolation/Color = %q/%q/%q", a.Effort, a.Isolation, a.Color)
	}
	if a.InitialPrompt != "Review the latest diff." {
		t.Errorf("InitialPrompt = %q", a.InitialPrompt)
	}
	if a.PluginDistributed {
		t.Errorf("PluginDistributed should be false straight from the parser")
	}
	if a.Frontmatter.KeyRange("maxTurns").IsZero() {
		t.Errorf("maxTurns key range should be recorded")
	}
}

// TestParseAgentAbsentFieldsZero pins absent-key zero values for the
// extended field set.
func TestParseAgentAbsentFieldsZero(t *testing.T) {
	src := []byte("---\nname: minimal\ndescription: d\n---\nbody\n")
	a, perr := ParseAgent(".claude/agents/minimal.md", src)
	if perr != nil {
		t.Fatalf("ParseAgent = %v", perr)
	}
	if a.Model != "" || a.PermissionMode != "" || a.MaxTurns != 0 {
		t.Errorf("scalars should be zero: %+v", a)
	}
	if a.HasMCPServers || a.HasHooks || a.Background {
		t.Errorf("presence bools should be false: %+v", a)
	}
	if a.DisallowedTools != nil || a.Skills != nil {
		t.Errorf("lists should be nil: %+v", a)
	}
}

// TestParseToolListStringForms covers the doc-canonical string forms:
// comma-separated agent tools and space/comma-separated allowed-tools,
// including permission-rule entries that must survive as one token.
func TestParseToolListStringForms(t *testing.T) {
	cmdSrc := []byte(
		"---\ndescription: x\nallowed-tools: Bash(git add:*) Bash(git status:*), Read\n---\n")
	c, perr := ParseCommand(".claude/commands/commit.md", cmdSrc)
	if perr != nil {
		t.Fatalf("ParseCommand = %v", perr)
	}
	want := []string{"Bash(git add:*)", "Bash(git status:*)", "Read"}
	if len(c.AllowedTools) != len(want) {
		t.Fatalf("AllowedTools = %v, want %v", c.AllowedTools, want)
	}
	for i := range want {
		if c.AllowedTools[i] != want[i] {
			t.Errorf("AllowedTools[%d] = %q, want %q", i, c.AllowedTools[i], want[i])
		}
	}

	agSrc := []byte("---\nname: helper\ndescription: y\ntools: Read, Grep, mcp__github\n---\n")
	a, perr := ParseAgent(".claude/agents/helper.md", agSrc)
	if perr != nil {
		t.Fatalf("ParseAgent = %v", perr)
	}
	if len(a.Tools) != 3 || a.Tools[0] != "Read" || a.Tools[1] != "Grep" || a.Tools[2] != "mcp__github" {
		t.Errorf("Tools = %v, want [Read Grep mcp__github]", a.Tools)
	}
}

func TestParseMarkdownUnterminatedFrontmatter(t *testing.T) {
	src := []byte("---\nname: writer\nbody continues forever")
	_, perr := parseMarkdown("x", src)
	if perr == nil {
		t.Fatal("expected ParseError, got nil")
	}
	if !strings.Contains(perr.Message, "unterminated") {
		t.Errorf("message = %q, want contains 'unterminated'", perr.Message)
	}
	if perr.Range.IsZero() {
		t.Errorf("range should point at opening fence, got zero")
	}
}

func TestParseMarkdownInvalidYAML(t *testing.T) {
	src := []byte("---\nname: [unbalanced\n---\n")
	_, perr := parseMarkdown("x", src)
	if perr == nil {
		t.Fatal("expected ParseError, got nil")
	}
	if !strings.Contains(perr.Message, "invalid YAML") {
		t.Errorf("message = %q, want contains 'invalid YAML'", perr.Message)
	}
}

func TestParseClaudeMDNoFrontmatter(t *testing.T) {
	src := []byte("# CLAUDE.md\n\ninstructions\n")
	c, perr := ParseClaudeMD("CLAUDE.md", src)
	if perr != nil {
		t.Fatalf("ParseClaudeMD = %v", perr)
	}
	if !c.Frontmatter.Block.IsZero() {
		t.Errorf("CLAUDE.md without frontmatter should have zero Block")
	}
	if !bytes.Equal(c.Source(), src) {
		t.Errorf("Source should equal input")
	}
}

func TestYAMLValueTypes(t *testing.T) {
	src := []byte(`---
str: hello
num: 42
flt: 3.14
flag: true
empty: null
nested:
  inner: value
list:
  - a
  - b
---
`)
	doc, err := parseMarkdown("x.md", src)
	if err != nil {
		t.Fatalf("parseMarkdown = %v", err)
	}
	if got := doc.fm["str"]; got != "hello" {
		t.Errorf("str = %v", got)
	}
	// Numeric / bool values are captured as their native Go type via
	// yamlValue; no coercion happens at this layer.
	if got, ok := doc.fm["num"].(uint64); !ok || got != 42 {
		t.Errorf("num = %v/%T", doc.fm["num"], doc.fm["num"])
	}
	if got, ok := doc.fm["flt"].(float64); !ok || got != 3.14 {
		t.Errorf("flt = %v/%T", doc.fm["flt"], doc.fm["flt"])
	}
	if got, ok := doc.fm["flag"].(bool); !ok || got != true {
		t.Errorf("flag = %v/%T", doc.fm["flag"], doc.fm["flag"])
	}
	if got := doc.fm["empty"]; got != nil {
		t.Errorf("empty = %v, want nil", got)
	}
	nested, ok := doc.fm["nested"].(map[string]any)
	if !ok || nested["inner"] != "value" {
		t.Errorf("nested = %v", doc.fm["nested"])
	}
	list, ok := doc.fm["list"].([]any)
	if !ok || len(list) != 2 {
		t.Errorf("list = %v", doc.fm["list"])
	}
}

func TestAsStringAndListEdgeCases(t *testing.T) {
	doc := &markdownDoc{fm: map[string]any{
		"nope":        42, // non-string
		"scalar":      "one",
		"real_list":   []any{"a", "b"},
		"single_list": "x", // coerced
		"weird":       42,  // not a string list either
	}}
	if doc.asString("missing") != "" {
		t.Errorf("asString missing should be empty")
	}
	if doc.asString("nope") != "" {
		t.Errorf("asString non-string should be empty")
	}
	if got := doc.asStringList("missing"); got != nil {
		t.Errorf("asStringList missing = %v, want nil", got)
	}
	if got := doc.asStringList("single_list"); len(got) != 1 || got[0] != "x" {
		t.Errorf("asStringList single = %v", got)
	}
	if got := doc.asStringList("weird"); got != nil {
		t.Errorf("asStringList non-list non-string = %v, want nil", got)
	}
}

func TestSplitToolList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"separators only", " , ,  ", nil},
		{"single tool", "Read", []string{"Read"}},
		{"comma separated", "Read,Write,Edit", []string{"Read", "Write", "Edit"}},
		{"comma space separated", "Read, Write, Edit", []string{"Read", "Write", "Edit"}},
		{"space separated", "Read Write Edit", []string{"Read", "Write", "Edit"}},
		{"mixed separators", "Read,  Write\tEdit", []string{"Read", "Write", "Edit"}},
		{
			"permission rule with space inside parens",
			"Bash(git add:*), Read",
			[]string{"Bash(git add:*)", "Read"},
		},
		{
			"comma inside parens",
			"Agent(worker, researcher) Read",
			[]string{"Agent(worker, researcher)", "Read"},
		},
		{
			"mcp patterns",
			"mcp__github mcp__linear__create_issue",
			[]string{"mcp__github", "mcp__linear__create_issue"},
		},
		{
			"nested parens",
			"Bash(echo $(date):*), Grep",
			[]string{"Bash(echo $(date):*)", "Grep"},
		},
		{
			"unbalanced open paren swallows rest",
			"Bash(git add Read",
			[]string{"Bash(git add Read"},
		},
		{
			"stray close paren stays attached",
			"Read) Write",
			[]string{"Read)", "Write"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitToolList(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("SplitToolList(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("SplitToolList(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestAsToolList(t *testing.T) {
	doc := &markdownDoc{fm: map[string]any{
		"str_form":  "Bash(git add:*), Read Edit",
		"list_form": []any{"Bash(git add:*)", "Read"},
	}}
	if got := doc.asToolList("missing"); got != nil {
		t.Errorf("asToolList missing = %v, want nil", got)
	}
	got := doc.asToolList("str_form")
	if len(got) != 3 || got[0] != "Bash(git add:*)" || got[1] != "Read" || got[2] != "Edit" {
		t.Errorf("asToolList string form = %v", got)
	}
	// YAML list elements pass through verbatim — never re-split.
	got = doc.asToolList("list_form")
	if len(got) != 2 || got[0] != "Bash(git add:*)" || got[1] != "Read" {
		t.Errorf("asToolList list form = %v", got)
	}
}

func TestParseMarkdownNonMappingFrontmatter(t *testing.T) {
	src := []byte("---\n- just\n- a\n- list\n---\nbody\n")
	_, perr := parseMarkdown("x.md", src)
	if perr == nil {
		t.Fatal("expected ParseError for non-mapping frontmatter")
	}
	if !strings.Contains(perr.Message, "mapping") {
		t.Errorf("message = %q, want contains 'mapping'", perr.Message)
	}
}

func TestParseMarkdownEmptyFrontmatter(t *testing.T) {
	src := []byte("---\n---\nbody\n")
	doc, perr := parseMarkdown("x.md", src)
	if perr != nil {
		t.Fatalf("parseMarkdown empty frontmatter = %v", perr)
	}
	if len(doc.Frontmatter.Keys) != 0 {
		t.Errorf("empty frontmatter should have no Keys, got %v", keysOf(doc.Frontmatter.Keys))
	}
}

func keysOf(m map[string]diag.Range) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
