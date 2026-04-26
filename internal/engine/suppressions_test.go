package engine

import (
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/config"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

// registerLineEmitter registers a stub rule that reports one
// diagnostic on the given line for every matching artifact. The line
// lets tests target the same line number that an in-source suppression
// marker is placed on.
func registerLineEmitter(t *testing.T, id string, line int, kinds ...artifact.ArtifactKind) {
	t.Helper()
	rules.Register(&stubRule{
		id:      id,
		sev:     diag.SeverityWarning,
		applies: kinds,
		checkFn: func(_ rules.Context, a artifact.Artifact) []diag.Diagnostic {
			return []diag.Diagnostic{{
				Path:    a.Path(),
				Range:   diag.Range{Start: diag.Position{Line: line}},
				Message: "flagged",
			}}
		},
	})
}

func TestSuppressPerLineMarker(t *testing.T) {
	rules.Reset()
	// Rule fires on line 3; suppression marker sits at line 3.
	registerLineEmitter(t, "a/x", 3, artifact.KindClaudeMD)

	src := []byte("# Heading\n\n<!-- claudelint:ignore=a/x --> some text\n")
	arts := []artifact.Artifact{&fakeArtifact{path: "CLAUDE.md", kind: artifact.KindClaudeMD, source: src}}

	res := New(nil).Run(arts, nil)
	if len(res.Diagnostics) != 0 {
		t.Errorf("per-line marker did not suppress: %+v", res.Diagnostics)
	}
}

func TestSuppressMarkerAboveNextLine(t *testing.T) {
	rules.Reset()
	// Marker on line 2 should silence a diagnostic reported on line 3
	// (the next non-blank line).
	registerLineEmitter(t, "a/x", 3, artifact.KindClaudeMD)

	src := []byte("# Heading\n<!-- claudelint:ignore=a/x -->\nflagged line\n")
	arts := []artifact.Artifact{&fakeArtifact{path: "CLAUDE.md", kind: artifact.KindClaudeMD, source: src}}

	res := New(nil).Run(arts, nil)
	if len(res.Diagnostics) != 0 {
		t.Errorf("above-marker did not suppress next line: %+v", res.Diagnostics)
	}
}

func TestSuppressPerFileMarker(t *testing.T) {
	rules.Reset()
	// Diagnostic on line 10 — per-file marker near top still silences.
	registerLineEmitter(t, "a/x", 10, artifact.KindSkill)

	src := []byte("---\nname: s\n---\n<!-- claudelint:ignore-file=a/x -->\n\n# body\n")
	arts := []artifact.Artifact{&fakeArtifact{path: "skills/s/SKILL.md", kind: artifact.KindSkill, source: src}}

	res := New(nil).Run(arts, nil)
	if len(res.Diagnostics) != 0 {
		t.Errorf("per-file marker did not suppress: %+v", res.Diagnostics)
	}
}

func TestSuppressPerFileOnlyTargetsListedIDs(t *testing.T) {
	rules.Reset()
	registerLineEmitter(t, "a/x", 1, artifact.KindClaudeMD)
	registerLineEmitter(t, "b/y", 1, artifact.KindClaudeMD)

	src := []byte("<!-- claudelint:ignore-file=a/x -->\nbody\n")
	arts := []artifact.Artifact{&fakeArtifact{path: "CLAUDE.md", kind: artifact.KindClaudeMD, source: src}}

	res := New(nil).Run(arts, nil)
	if len(res.Diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want exactly b/y: %+v", len(res.Diagnostics), res.Diagnostics)
	}
	if res.Diagnostics[0].RuleID != "b/y" {
		t.Errorf("surviving RuleID = %q, want b/y", res.Diagnostics[0].RuleID)
	}
}

func TestSuppressInSourceIgnoredForJSONKinds(t *testing.T) {
	rules.Reset()
	// Hook/Plugin are JSON-backed; the marker should NOT take effect
	// even though it textually appears in the source.
	registerLineEmitter(t, "hooks/z", 1, artifact.KindHook)

	src := []byte(`{"hooks": "<!-- claudelint:ignore-file=hooks/z -->"}`)
	arts := []artifact.Artifact{&fakeArtifact{path: ".claude/hooks.json", kind: artifact.KindHook, source: src}}

	res := New(nil).Run(arts, nil)
	if len(res.Diagnostics) != 1 {
		t.Errorf("JSON kind should ignore in-source markers; got %+v", res.Diagnostics)
	}
}

// TestSuppressSecretsByIDPerLineMarker is the integration regression
// for issue #15: with security/secrets now emitting a real
// Range.Start.Line, the engine's per-line suppressor can match the
// marker `<!-- claudelint:ignore=security/secrets -->` on the line
// above the offending token. We register a stub rule under the
// real security/secrets ID to keep this test isolated from the
// rule's own unit tests (which prove the Range itself is correct).
func TestSuppressSecretsByIDPerLineMarker(t *testing.T) {
	rules.Reset()
	registerLineEmitter(t, "security/secrets", 4, artifact.KindClaudeMD)

	src := []byte(
		"# Title\n" +
			"\n" +
			"<!-- claudelint:ignore=security/secrets -->\n" +
			"key: sk-ABCDEFGHIJKLMNOPQRSTUV12345\n",
	)
	arts := []artifact.Artifact{&fakeArtifact{
		path:   "CLAUDE.md",
		kind:   artifact.KindClaudeMD,
		source: src,
	}}

	res := New(nil).Run(arts, nil)
	if len(res.Diagnostics) != 0 {
		t.Errorf("per-line marker did not suppress security/secrets: %+v", res.Diagnostics)
	}
	if len(res.Suppressed) != 1 {
		t.Errorf("expected 1 suppressed diagnostic, got %d", len(res.Suppressed))
	}
}

// TestSuppressionMechanismMatrix verifies that each of the three
// suppression mechanisms independently silences a rule: config-level
// enabled=false, config-level per-rule paths glob, and in-source
// marker. They are assembled in the same order DESIGN-0001 documents.
func TestSuppressionMechanismMatrix(t *testing.T) {
	cases := []struct {
		name   string
		cfg    *config.File
		source []byte
	}{
		{
			name: "enabled=false",
			cfg: &config.File{
				Claudelint: &config.Claudelint{Version: "1"},
				Rules:      []config.RuleBlock{{ID: "a/x", Enabled: boolPtr(false)}},
			},
		},
		{
			name: "per-rule-paths-glob",
			cfg: &config.File{
				Claudelint: &config.Claudelint{Version: "1"},
				Rules:      []config.RuleBlock{{ID: "a/x", Paths: []string{"CLAUDE.md"}}},
			},
		},
		{
			name:   "in-source-ignore-file",
			cfg:    nil,
			source: []byte("<!-- claudelint:ignore-file=a/x -->\nbody\n"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rules.Reset()
			registerLineEmitter(t, "a/x", 2, artifact.KindClaudeMD)

			arts := []artifact.Artifact{&fakeArtifact{
				path:   "CLAUDE.md",
				kind:   artifact.KindClaudeMD,
				source: tc.source,
			}}
			rc := config.Resolve(tc.cfg)
			res := New(rc).Run(arts, nil)
			if len(res.Diagnostics) != 0 {
				t.Errorf("%s did not suppress a/x: %+v", tc.name, res.Diagnostics)
			}
		})
	}
}

func TestUnknownRuleEmitsMetaWarning(t *testing.T) {
	rules.Reset()
	// Register the real rule; user misspells it in config. The real
	// rule should continue to fire; a meta/unknown-rule warning should
	// surface the typo.
	registerLineEmitter(t, "a/x", 1, artifact.KindClaudeMD)

	cfg := config.Resolve(&config.File{
		Claudelint: &config.Claudelint{Version: "1"},
		Rules: []config.RuleBlock{
			{ID: "a/xtypo", Enabled: boolPtr(false)},
		},
	}).WithPath("/tmp/.claudelint.hcl")

	arts := []artifact.Artifact{&fakeArtifact{path: "CLAUDE.md", kind: artifact.KindClaudeMD}}
	res := New(cfg).Run(arts, nil)

	var sawMeta, sawReal bool
	for _, d := range res.Diagnostics {
		switch d.RuleID {
		case MetaUnknownRuleID:
			sawMeta = true
			if d.Severity != diag.SeverityWarning {
				t.Errorf("meta severity = %v, want warning", d.Severity)
			}
			if d.Path != "/tmp/.claudelint.hcl" {
				t.Errorf("meta path = %q, want config path", d.Path)
			}
		case "a/x":
			sawReal = true
		}
	}
	if !sawMeta {
		t.Errorf("expected meta/unknown-rule diagnostic, got %+v", res.Diagnostics)
	}
	if !sawReal {
		t.Errorf("typo'd config ID should not disable a/x; got %+v", res.Diagnostics)
	}
}

func boolPtr(b bool) *bool { return &b }
