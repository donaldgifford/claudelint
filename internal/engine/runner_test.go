package engine

import (
	"sync/atomic"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/config"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// stubRule lets tests construct rules inline without depending on the
// real MVP rule set.
type stubRule struct {
	id       string
	cat      string
	sev      diag.Severity
	opts     map[string]any
	applies  []artifact.ArtifactKind
	checkFn  func(rules.Context, artifact.Artifact) []diag.Diagnostic
	callsCtr *int64
}

func (s *stubRule) ID() string                         { return s.id }
func (s *stubRule) Category() string                   { return s.cat }
func (s *stubRule) DefaultSeverity() diag.Severity     { return s.sev }
func (s *stubRule) DefaultOptions() map[string]any     { return s.opts }
func (s *stubRule) AppliesTo() []artifact.ArtifactKind { return s.applies }
func (s *stubRule) Check(ctx rules.Context, a artifact.Artifact) []diag.Diagnostic {
	if s.callsCtr != nil {
		atomic.AddInt64(s.callsCtr, 1)
	}
	if s.checkFn == nil {
		return nil
	}
	return s.checkFn(ctx, a)
}

// fakeArtifact implements artifact.Artifact without requiring a parse.
type fakeArtifact struct {
	path string
	kind artifact.ArtifactKind
}

func (f *fakeArtifact) Kind() artifact.ArtifactKind { return f.kind }
func (f *fakeArtifact) Path() string                { return f.path }
func (*fakeArtifact) Source() []byte                { return nil }

func TestRunNoRulesNoArtifacts(t *testing.T) {
	rules.Reset()
	res := New(nil).Run(nil, nil)
	if res.Files != 0 || len(res.Diagnostics) != 0 {
		t.Errorf("empty run = %+v", res)
	}
}

func TestRunAppliesRuleToMatchingKind(t *testing.T) {
	rules.Reset()
	calls := int64(0)
	rules.Register(&stubRule{
		id:       "skills/require-name",
		sev:      diag.SeverityError,
		applies:  []artifact.ArtifactKind{artifact.KindSkill},
		callsCtr: &calls,
		checkFn: func(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
			return []diag.Diagnostic{{
				Path:    a.Path(),
				Message: "missing name",
			}}
		},
	})
	// Register a rule that targets a different kind — should not fire.
	rules.Register(&stubRule{
		id:      "commands/other",
		sev:     diag.SeverityWarning,
		applies: []artifact.ArtifactKind{artifact.KindCommand},
	})

	arts := []artifact.Artifact{
		&fakeArtifact{path: "skills/a/SKILL.md", kind: artifact.KindSkill},
		&fakeArtifact{path: "skills/b/SKILL.md", kind: artifact.KindSkill},
	}
	res := New(nil).Run(arts, nil)
	if res.Files != 2 {
		t.Errorf("Files = %d, want 2", res.Files)
	}
	if len(res.Diagnostics) != 2 {
		t.Errorf("diagnostics = %d, want 2", len(res.Diagnostics))
	}
	for _, d := range res.Diagnostics {
		if d.RuleID != "skills/require-name" {
			t.Errorf("RuleID = %q", d.RuleID)
		}
		if d.Severity != diag.SeverityError {
			t.Errorf("Severity = %v", d.Severity)
		}
	}
	if calls != 2 {
		t.Errorf("rule Check called %d times, want 2", calls)
	}
}

func TestRunSynthesizesParseDiagnostic(t *testing.T) {
	rules.Reset()
	pe := &artifact.ParseError{
		Path:    "bad/x.md",
		Message: "unterminated YAML frontmatter",
	}
	res := New(nil).Run(nil, []*artifact.ParseError{pe})
	if len(res.Diagnostics) != 1 {
		t.Fatalf("expected 1 synthesized diagnostic, got %d", len(res.Diagnostics))
	}
	d := res.Diagnostics[0]
	if d.RuleID != ParseSchemaRuleID {
		t.Errorf("RuleID = %q, want %q", d.RuleID, ParseSchemaRuleID)
	}
	if d.Severity != diag.SeverityError {
		t.Errorf("severity = %v, want error", d.Severity)
	}
	if d.Message != "unterminated YAML frontmatter" {
		t.Errorf("message = %q", d.Message)
	}
}

func TestRunSortsDiagnostics(t *testing.T) {
	rules.Reset()
	rules.Register(&stubRule{
		id:      "x/report",
		sev:     diag.SeverityWarning,
		applies: []artifact.ArtifactKind{artifact.KindClaudeMD},
		checkFn: func(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
			return []diag.Diagnostic{
				{Path: a.Path(), Range: diag.Range{Start: diag.Position{Line: 5}}, Message: "later"},
				{Path: a.Path(), Range: diag.Range{Start: diag.Position{Line: 1}}, Message: "earlier"},
			}
		},
	})

	arts := []artifact.Artifact{
		&fakeArtifact{path: "b.md", kind: artifact.KindClaudeMD},
		&fakeArtifact{path: "a.md", kind: artifact.KindClaudeMD},
	}
	res := New(nil).Run(arts, nil)
	// a.md ordering: two diagnostics, sorted by line; b.md follows.
	wantPaths := []string{"a.md", "a.md", "b.md", "b.md"}
	for i, d := range res.Diagnostics {
		if d.Path != wantPaths[i] {
			t.Errorf("diag[%d].Path = %q, want %q", i, d.Path, wantPaths[i])
		}
	}
	if res.Diagnostics[0].Range.Start.Line != 1 || res.Diagnostics[1].Range.Start.Line != 5 {
		t.Errorf("a.md not line-sorted: %+v", res.Diagnostics[:2])
	}
}

func TestRunDedupesIdenticalDiagnostics(t *testing.T) {
	rules.Reset()
	// Two rules that emit identical diagnostics — dedup collapses to one.
	for _, id := range []string{"a/x", "b/x"} {
		rules.Register(&stubRule{
			id:      id,
			sev:     diag.SeverityWarning,
			applies: []artifact.ArtifactKind{artifact.KindClaudeMD},
			checkFn: func(_ rules.Context, _ artifact.Artifact) []diag.Diagnostic {
				return []diag.Diagnostic{{
					Path:    "x.md",
					Message: "same message",
				}}
			},
		})
	}
	arts := []artifact.Artifact{&fakeArtifact{path: "x.md", kind: artifact.KindClaudeMD}}
	res := New(nil).Run(arts, nil)
	// Both rules emit "same message" but with different RuleIDs, so
	// they should NOT dedupe. Dedup is exact-equality only.
	if len(res.Diagnostics) != 2 {
		t.Errorf("unique-by-RuleID diagnostics = %d, want 2", len(res.Diagnostics))
	}
}

func TestRunConcurrencyRaceClean(t *testing.T) {
	rules.Reset()
	rules.Register(&stubRule{
		id:      "style/noop",
		sev:     diag.SeverityInfo,
		applies: []artifact.ArtifactKind{artifact.KindSkill},
		checkFn: func(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
			return []diag.Diagnostic{{Path: a.Path(), Message: "hi"}}
		},
	})

	const n = 200
	arts := make([]artifact.Artifact, n)
	for i := range arts {
		arts[i] = &fakeArtifact{path: "s.md", kind: artifact.KindSkill}
	}
	res := New(nil, WithWorkers(8)).Run(arts, nil)
	// 200 identical diagnostics dedupe to 1 after sort.
	if len(res.Diagnostics) != 1 {
		t.Errorf("parallel run diagnostics = %d, want 1 after dedupe", len(res.Diagnostics))
	}
}

func TestRunRespectsRuleEnabledFalse(t *testing.T) {
	rules.Reset()
	rules.Register(&stubRule{
		id:      "a/x",
		sev:     diag.SeverityError,
		applies: []artifact.ArtifactKind{artifact.KindSkill},
		checkFn: func(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
			return []diag.Diagnostic{{Path: a.Path(), Message: "hi"}}
		},
	})

	// Craft a ResolvedConfig that disables a/x.
	disabled := false
	cfg := config.Resolve(&config.File{
		Claudelint: &config.Claudelint{Version: "1"},
		Rules:      []config.RuleBlock{{ID: "a/x", Enabled: &[]bool{disabled}[0]}},
	})

	arts := []artifact.Artifact{&fakeArtifact{path: "x.md", kind: artifact.KindSkill}}
	res := New(cfg).Run(arts, nil)
	if len(res.Diagnostics) != 0 {
		t.Errorf("disabled rule still emitted %d diagnostics", len(res.Diagnostics))
	}
}

func TestRunOptionOverlay(t *testing.T) {
	rules.Reset()
	var observed atomic.Value
	rules.Register(&stubRule{
		id:      "a/opt",
		sev:     diag.SeverityWarning,
		applies: []artifact.ArtifactKind{artifact.KindSkill},
		opts:    map[string]any{"max_words": 1000},
		checkFn: func(c rules.Context, _ artifact.Artifact) []diag.Diagnostic {
			observed.Store(c.Option("max_words"))
			return nil
		},
	})

	cfg := config.Resolve(&config.File{
		Claudelint: &config.Claudelint{Version: "1"},
	})
	_ = New(cfg).Run(
		[]artifact.Artifact{&fakeArtifact{path: "x.md", kind: artifact.KindSkill}},
		nil,
	)
	if got := observed.Load(); got != 1000 {
		t.Errorf("Option(max_words) = %v, want default 1000", got)
	}
}
