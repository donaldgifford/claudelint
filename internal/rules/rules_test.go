package rules

import (
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
)

// stubRule is a minimal Rule used by registry and fingerprint tests.
type stubRule struct {
	id      string
	cat     string
	sev     diag.Severity
	opts    map[string]any
	applies []artifact.ArtifactKind
	checkFn func(Context, artifact.Artifact) []diag.Diagnostic
}

func (s *stubRule) ID() string                         { return s.id }
func (s *stubRule) Category() string                   { return s.cat }
func (s *stubRule) DefaultSeverity() diag.Severity     { return s.sev }
func (s *stubRule) DefaultOptions() map[string]any     { return s.opts }
func (s *stubRule) AppliesTo() []artifact.ArtifactKind { return s.applies }
func (s *stubRule) HelpURI() string                    { return DefaultHelpURI(s.id) }
func (s *stubRule) Check(ctx Context, a artifact.Artifact) []diag.Diagnostic {
	if s.checkFn != nil {
		return s.checkFn(ctx, a)
	}
	return nil
}

func TestRegisterAndAll(t *testing.T) {
	Reset()
	Register(&stubRule{id: "b/x", cat: "style"})
	Register(&stubRule{id: "a/x", cat: "schema"})

	got := All()
	if len(got) != 2 {
		t.Fatalf("All() len = %d, want 2", len(got))
	}
	if got[0].ID() != "a/x" || got[1].ID() != "b/x" {
		t.Errorf("All() IDs = [%s, %s], want sorted", got[0].ID(), got[1].ID())
	}
}

func TestGet(t *testing.T) {
	Reset()
	Register(&stubRule{id: "a/x"})
	if r := Get("a/x"); r == nil || r.ID() != "a/x" {
		t.Errorf("Get known = %v, want stub", r)
	}
	if r := Get("missing"); r != nil {
		t.Errorf("Get missing = %v, want nil", r)
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	Reset()
	Register(&stubRule{id: "a/x"})
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("duplicate Register did not panic")
		} else if !strings.Contains(r.(string), `"a/x"`) {
			t.Errorf("panic message = %v, want contains duplicate id", r)
		}
	}()
	Register(&stubRule{id: "a/x"})
}

func TestRegisterNilPanics(t *testing.T) {
	Reset()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("nil Register did not panic")
		}
	}()
	Register(nil)
}

func TestRegisterEmptyIDPanics(t *testing.T) {
	Reset()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("empty ID Register did not panic")
		}
	}()
	Register(&stubRule{id: ""})
}

func TestRulesetFingerprintStable(t *testing.T) {
	Reset()
	Register(
		&stubRule{id: "x/1", cat: "c", sev: diag.SeverityWarning, opts: map[string]any{"k": 1}, applies: []artifact.ArtifactKind{artifact.KindSkill}},
	)
	Register(&stubRule{id: "y/2", cat: "c", sev: diag.SeverityError})

	a := RulesetFingerprint()
	b := RulesetFingerprint()
	if a != b {
		t.Errorf("RulesetFingerprint changes across calls: %q vs %q", a, b)
	}
	if len(a) != 8 {
		t.Errorf("fingerprint length = %d, want 8", len(a))
	}
}

func TestRulesetFingerprintChangesWithRuleset(t *testing.T) {
	Reset()
	Register(&stubRule{id: "x/1", cat: "c", sev: diag.SeverityWarning})
	before := RulesetFingerprint()

	Register(&stubRule{id: "y/2", cat: "c", sev: diag.SeverityError})
	after := RulesetFingerprint()

	if before == after {
		t.Errorf("fingerprint should change when a rule is added")
	}
}

func TestRulesetFingerprintChangesOnDefaultSeverity(t *testing.T) {
	Reset()
	Register(&stubRule{id: "x/1", cat: "c", sev: diag.SeverityWarning})
	a := RulesetFingerprint()

	Reset()
	Register(&stubRule{id: "x/1", cat: "c", sev: diag.SeverityError})
	b := RulesetFingerprint()

	if a == b {
		t.Errorf("fingerprint should differ when default severity changes")
	}
}
