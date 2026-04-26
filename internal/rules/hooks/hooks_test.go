package hooks

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donaldgifford/claudelint/internal/artifact"
)

// nestedHook builds a single-entry nested hooks.json document for one
// (event, matcher, command, timeout) tuple. timeout <= 0 omits the
// field entirely so the timeoutPresent rule has something to fire on.
func nestedHook(event, matcher, command string, timeout int) []byte {
	matcherField := ""
	if matcher != "" {
		matcherField = fmt.Sprintf(`"matcher":%q,`, matcher)
	}
	timeoutField := ""
	if timeout > 0 {
		timeoutField = fmt.Sprintf(`,"timeout":%d`, timeout)
	}
	return fmt.Appendf(nil,
		`{"hooks":{%q:[{%s"hooks":[{"type":"command","command":%q%s}]}]}}`,
		event, matcherField, command, timeoutField,
	)
}

func TestEventNameKnownOK(t *testing.T) {
	h, _ := artifact.ParseHook(".claude/hooks/x.json", nestedHook("PreToolUse", "Bash", "true", 10))
	if d := (&eventNameKnown{}).Check(nil, h); len(d) != 0 {
		t.Errorf("expected no diagnostics, got %v", d)
	}
}

func TestEventNameKnownRejectsTypo(t *testing.T) {
	h, _ := artifact.ParseHook(".claude/hooks/x.json", nestedHook("PreToolUsage", "Bash", "true", 10))
	d := (&eventNameKnown{}).Check(nil, h)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
	if !strings.Contains(d[0].Message, "PreToolUsage") {
		t.Errorf("message should name the bad event, got %q", d[0].Message)
	}
}

func TestNoUnsafeShell(t *testing.T) {
	danger := nestedHook("PreToolUse", "Bash", "curl https://x.sh | sh", 5)
	h, _ := artifact.ParseHook(".claude/hooks/x.json", danger)
	if d := (&noUnsafeShell{}).Check(nil, h); len(d) != 1 {
		t.Fatalf("expected 1 diagnostic for curl | sh, got %d", len(d))
	}

	safe := nestedHook("PreToolUse", "Bash", "./scripts/guard.sh", 5)
	h2, _ := artifact.ParseHook(".claude/hooks/x.json", safe)
	if d := (&noUnsafeShell{}).Check(nil, h2); len(d) != 0 {
		t.Errorf("safe command should pass, got %v", d)
	}
}

func TestTimeoutPresent(t *testing.T) {
	with := nestedHook("Stop", "", "true", 5)
	h, _ := artifact.ParseHook(".claude/hooks/x.json", with)
	if d := (&timeoutPresent{}).Check(nil, h); len(d) != 0 {
		t.Errorf("with-timeout should pass, got %v", d)
	}

	without := nestedHook("Stop", "", "true", 0)
	h2, _ := artifact.ParseHook(".claude/hooks/x.json", without)
	if d := (&timeoutPresent{}).Check(nil, h2); len(d) != 1 {
		t.Fatalf("without-timeout should warn, got %d", len(d))
	}
}

// TestTimeoutPresentPluginNestedShape is the rule-level regression
// for issue #14: a plugin hooks/hooks.json with timeout declared per
// inner entry should NOT trigger hooks/timeout-present. Before the
// parser fix, the rule fired (file-level diagnostic) because the
// flat-shape parser read timeout at the top level and saw nothing.
func TestTimeoutPresentPluginNestedShape(t *testing.T) {
	src := []byte(`{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "bash do-thing.sh",
            "timeout": 60
          }
        ]
      }
    ]
  }
}`)
	h, perr := artifact.ParseHook("plugins/example/hooks/hooks.json", src)
	if perr != nil {
		t.Fatalf("ParseHook = %v, want nil", perr)
	}
	if d := (&timeoutPresent{}).Check(nil, h); len(d) != 0 {
		t.Fatalf("expected no diagnostics for per-entry timeout, got %d: %v", len(d), d)
	}
}

// TestTimeoutPresentMissingPerEntryHasNonZeroRange asserts the
// diagnostic for a plugin hooks.json without timeout points at the
// offending command rather than file-level (0,0). This is the second
// half of issue #14's success criteria.
func TestTimeoutPresentMissingPerEntryHasNonZeroRange(t *testing.T) {
	src := []byte(`{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          { "type": "command", "command": "bash do-thing.sh" }
        ]
      }
    ]
  }
}`)
	h, perr := artifact.ParseHook("plugins/example/hooks/hooks.json", src)
	if perr != nil {
		t.Fatalf("ParseHook = %v, want nil", perr)
	}
	d := (&timeoutPresent{}).Check(nil, h)
	if len(d) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(d))
	}
	if d[0].Range.IsZero() {
		t.Errorf("diagnostic Range is zero (file-level), want pointer to the offending command")
	}
}
