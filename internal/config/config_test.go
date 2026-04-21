package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/diag"
)

func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, Filename)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

func TestLoadOK(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
claudelint { version = "1" }

rule "skills/body-size" {
  enabled  = true
  severity = "error"
  options  = { max_words = 1500 }
  paths    = ["legacy/**"]
}

rules "skill" {
  default_severity = "warning"
}

ignore {
  paths = ["vendor/**"]
}

output {
  format = "json"
}
`)

	res, err := Load("", dir)
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	if res == nil {
		t.Fatal("Load returned nil result")
	}
	if res.File.Claudelint.Version != "1" {
		t.Errorf("version = %q", res.File.Claudelint.Version)
	}

	rc := Resolve(res.File)
	if !rc.RuleEnabled("skills/body-size") {
		t.Errorf("rule should be enabled")
	}
	if rc.RuleSeverity("skills/body-size", "skill", diag.SeverityInfo) != diag.SeverityError {
		t.Errorf("severity should be error")
	}
	if rc.RuleSeverity("skills/unset", "skill", diag.SeverityInfo) != diag.SeverityWarning {
		t.Errorf("per-kind default should apply when rule-level is unset")
	}
	if v := rc.RuleOption("skills/body-size", "skill", "max_words", 1000); v != int64(1500) {
		t.Errorf("option = %v (%T), want 1500", v, v)
	}
	if !rc.PathIgnored("vendor/x.md") {
		t.Errorf("vendor/** should ignore vendor/x.md")
	}
	if !rc.PathIgnoredForRule("skills/body-size", "legacy/old.md") {
		t.Errorf("legacy/** should suppress skills/body-size in legacy/old.md")
	}
	if rc.Output() != "json" {
		t.Errorf("output = %q", rc.Output())
	}
}

func TestLoadMissingConfigReturnsNil(t *testing.T) {
	dir := t.TempDir()
	res, err := Load("", dir)
	if err != nil {
		t.Fatalf("Load = %v, want nil", err)
	}
	if res != nil {
		t.Fatalf("Load with no config = %+v, want nil", res)
	}
}

func TestLoadRejectsUnknownVersion(t *testing.T) {
	dir := t.TempDir()
	p := writeConfig(t, dir, `claudelint { version = "99" }`)
	_, err := Load(p, dir)
	if err == nil {
		t.Fatal("expected error for unknown schema version")
	}
	if !strings.Contains(err.Error(), "schema version") {
		t.Errorf("error = %q, want mentions schema version", err)
	}
}

func TestLoadRejectsMissingClaudelintBlock(t *testing.T) {
	dir := t.TempDir()
	p := writeConfig(t, dir, `output { format = "text" }`)
	_, err := Load(p, dir)
	if err == nil {
		t.Fatal("expected error for missing claudelint block")
	}
}

func TestLoadRejectsBadSeverity(t *testing.T) {
	dir := t.TempDir()
	p := writeConfig(t, dir, `
claudelint { version = "1" }
rule "x" { severity = "critical" }
`)
	_, err := Load(p, dir)
	if err == nil {
		t.Fatal("expected severity validation error")
	}
	if !strings.Contains(err.Error(), "critical") {
		t.Errorf("error = %q, want mentions bad value", err)
	}
}

func TestLoadRejectsBadFormat(t *testing.T) {
	dir := t.TempDir()
	p := writeConfig(t, dir, `
claudelint { version = "1" }
output { format = "xml" }
`)
	_, err := Load(p, dir)
	if err == nil {
		t.Fatal("expected format validation error")
	}
}

func TestLoadHCLSyntaxError(t *testing.T) {
	dir := t.TempDir()
	// Line-2 error so we can check the line number is non-trivial.
	p := writeConfig(t, dir, "claudelint { version = \"1\" }\noutput { format = }\n")
	_, err := Load(p, dir)
	if err == nil {
		t.Fatal("expected HCL syntax error")
	}
	// HCL's diagnostic format is "<path>:<line>,<col1>-<col2>: message".
	if !strings.Contains(err.Error(), ":2,") {
		t.Errorf("error should point at line 2, got %q", err.Error())
	}
}

func TestLoadBadOptionsType(t *testing.T) {
	dir := t.TempDir()
	// Options must be an object; a scalar is rejected at validation
	// time so the error names the offending rule.
	p := writeConfig(t, dir, `
claudelint { version = "1" }
rule "x" {
  options = "not-an-object"
}
`)
	_, err := Load(p, dir)
	if err == nil {
		t.Fatal("expected validation error for scalar options")
	}
	if !strings.Contains(err.Error(), "options must be an object") {
		t.Errorf("error = %q, want contains 'options must be an object'", err.Error())
	}
	if !strings.Contains(err.Error(), `"x"`) {
		t.Errorf("error should name the offending rule, got %q", err.Error())
	}
}

func TestFindConfigWalksUp(t *testing.T) {
	root := t.TempDir()
	// Put config at root, start from a nested dir.
	writeConfig(t, root, `claudelint { version = "1" }`)
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	found, err := FindConfig(nested)
	if err != nil {
		t.Fatalf("FindConfig = %v", err)
	}
	want := filepath.Join(root, Filename)
	if found != want {
		t.Errorf("FindConfig = %q, want %q", found, want)
	}
}

func TestResolveNilFile(t *testing.T) {
	rc := Resolve(nil)
	if !rc.RuleEnabled("anything") {
		t.Errorf("rules should default enabled with nil file")
	}
	if rc.Output() != "" {
		t.Errorf("Output should be empty with nil file")
	}
}

func TestResolveWithOptionTypes(t *testing.T) {
	dir := t.TempDir()
	p := writeConfig(t, dir, `
claudelint { version = "1" }

rule "a/b" {
  options = {
    max_words = 1200
    enable    = true
    name      = "mine"
    factor    = 1.5
    tags      = ["x", "y"]
    nested    = { k = "v" }
  }
}

rules "skill" {
  options = {
    shared_opt = "kind-level"
  }
}
`)
	res, err := Load(p, dir)
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	rc := Resolve(res.File)

	// Per-rule options are parsed into Go-native values by ctyToGo.
	if got := rc.RuleOption("a/b", "skill", "max_words", 0); got != int64(1200) {
		t.Errorf("max_words = %v (%T), want int64 1200", got, got)
	}
	if got := rc.RuleOption("a/b", "skill", "enable", false); got != true {
		t.Errorf("enable = %v, want true", got)
	}
	if got := rc.RuleOption("a/b", "skill", "name", ""); got != "mine" {
		t.Errorf("name = %v, want mine", got)
	}
	if got := rc.RuleOption("a/b", "skill", "factor", 0.0); got != 1.5 {
		t.Errorf("factor = %v, want 1.5", got)
	}
	tags, _ := rc.RuleOption("a/b", "skill", "tags", []any(nil)).([]any)
	if len(tags) != 2 || tags[0] != "x" {
		t.Errorf("tags = %v", tags)
	}
	nested, _ := rc.RuleOption("a/b", "skill", "nested", nil).(map[string]any)
	if nested["k"] != "v" {
		t.Errorf("nested = %v", nested)
	}

	// Per-kind shared_opt falls through when the rule block does not
	// override it.
	if got := rc.RuleOption("other/rule", "skill", "shared_opt", ""); got != "kind-level" {
		t.Errorf("kind-level shared_opt = %v", got)
	}

	// Default fallback when neither rule nor kind define the option.
	if got := rc.RuleOption("other/rule", "skill", "truly_missing", "fallback"); got != "fallback" {
		t.Errorf("fallback = %v", got)
	}
}

func TestResolveDisabledRule(t *testing.T) {
	dir := t.TempDir()
	p := writeConfig(t, dir, `
claudelint { version = "1" }
rule "x" { enabled = false }
`)
	res, err := Load(p, dir)
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	rc := Resolve(res.File)
	if rc.RuleEnabled("x") {
		t.Errorf("rule x should be disabled")
	}
	if !rc.RuleEnabled("y") {
		t.Errorf("rule y should default enabled")
	}
}

func TestPathIgnoredSimpleGlob(t *testing.T) {
	rc := Resolve(&File{
		Claudelint: &Claudelint{Version: "1"},
		Ignore:     &IgnoreBlock{Paths: []string{"node_modules/*"}},
	})
	if !rc.PathIgnored("node_modules/x") {
		t.Errorf("simple glob should match")
	}
	if rc.PathIgnored("src/x") {
		t.Errorf("simple glob should not match unrelated path")
	}
}
