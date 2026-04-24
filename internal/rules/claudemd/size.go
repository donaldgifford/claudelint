// Package claudemd holds rules that apply to CLAUDE.md artifacts.
package claudemd

import (
	"bytes"
	"fmt"

	"github.com/donaldgifford/claudelint/internal/artifact"
	"github.com/donaldgifford/claudelint/internal/diag"
	"github.com/donaldgifford/claudelint/internal/rules"
)

func init() { rules.Register(&size{}) }

const defaultMaxLines = 500

// size warns when a CLAUDE.md file exceeds a configurable line count.
// CLAUDE.md is loaded into every prompt; long files waste context
// and often indicate the file is duplicating content that belongs in
// a skill or plan doc.
type size struct{}

func (*size) ID() string                     { return "claude_md/size" }
func (*size) Category() string               { return "content" }
func (*size) DefaultSeverity() diag.Severity { return diag.SeverityWarning }
func (*size) DefaultOptions() map[string]any { return map[string]any{"max_lines": defaultMaxLines} }
func (*size) AppliesTo() []artifact.ArtifactKind {
	return []artifact.ArtifactKind{artifact.KindClaudeMD}
}

func (*size) HelpURI() string { return rules.DefaultHelpURI("claude_md/size") }

func (r *size) Check(ctx rules.Context, a artifact.Artifact) []diag.Diagnostic {
	c, ok := a.(*artifact.ClaudeMD)
	if !ok {
		return nil
	}
	limit := intOpt(ctx, "max_lines", defaultMaxLines)
	lines := bytes.Count(c.Source(), []byte("\n"))
	if lines <= limit {
		return nil
	}
	return []diag.Diagnostic{{
		RuleID:  r.ID(),
		Path:    c.Path(),
		Message: fmt.Sprintf("CLAUDE.md has %d lines; max is %d", lines, limit),
	}}
}

func intOpt(ctx rules.Context, key string, def int) int {
	switch v := ctx.Option(key).(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return def
	}
}
