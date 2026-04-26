package artifact

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"

	"github.com/buger/jsonparser"

	"github.com/donaldgifford/claudelint/internal/diag"
)

// ParseHook parses a hook declaration. claudelint accepts a single
// canonical shape regardless of which file it appears in:
//
//	{
//	  "hooks": {
//	    "<EventName>": [
//	      { "matcher": "<pattern>", "hooks": [
//	          { "type": "command", "command": "<cmd>", "timeout": <s> }
//	      ] }
//	    ]
//	  }
//	}
//
// This is the shape Claude Code documents for both
// .claude/settings{,.local}.json and plugin hooks/hooks.json (see
// https://docs.claude.com/en/docs/claude-code/hooks). Discovery also
// classifies .claude/hooks/*.json as KindHook even though that path
// is not canonically documented; we accept it on a best-effort basis
// using the same nested shape.
//
// Hook.Embedded is true when the source file is a settings.json (the
// hooks share a file with other Claude Code config) and false for
// dedicated hook files.
//
// Failure modes:
//   - syntactically invalid JSON → *ParseError
//   - "hooks" key present but not an object → *ParseError
//   - dedicated hook file missing the "hooks" key entirely → *ParseError
//     (settings files are allowed to omit it; the file may carry no hooks)
//
// Earlier versions of the parser also accepted a flat
// {event, matcher, command, timeout} shape for dedicated hook files.
// That shape never appeared in Claude Code's documentation and produced
// silent zero-Timeout values when the real nested shape was used (see
// issue #14). It has been removed; an unknown shape now fails loudly.
func ParseHook(path string, src []byte) (*Hook, *ParseError) {
	base := NewBase(path, src)

	if err := validateJSON(src); err != nil {
		return nil, &ParseError{
			Path:    path,
			Range:   base.ResolveRange(0, len(src)),
			Message: fmt.Sprintf("invalid JSON: %s", err.Error()),
			Cause:   err,
		}
	}

	h := &Hook{Base: base}
	settings := isSettingsFile(path)
	h.Embedded = settings

	missingHooksKey, err := collectHooks(src, &base, h)
	if err != nil {
		return nil, &ParseError{
			Path:    path,
			Range:   base.ResolveRange(0, len(src)),
			Message: err.Error(),
			Cause:   err,
		}
	}
	if missingHooksKey && !settings {
		return nil, &ParseError{
			Path:    path,
			Range:   base.ResolveRange(0, len(src)),
			Message: `hook file is missing the top-level "hooks" key; expected {"hooks": {"<EventName>": [{"hooks": [{"command": "...", "timeout": <s>}]}]}}`,
		}
	}
	return h, nil
}

// ParsePlugin parses a plugin manifest (plugin.json or plugin.yaml/yml).
// Only JSON is supported for v1; YAML manifests yield a ParseError
// asking the user to convert. The YAML path lands in phase 2 when the
// conversion subcommand work begins.
func ParsePlugin(path string, src []byte) (*Plugin, *ParseError) {
	base := NewBase(path, src)

	if !isJSONFile(path) {
		return nil, &ParseError{
			Path:    path,
			Range:   base.ResolveRange(0, len(src)),
			Message: "YAML plugin manifests are not supported yet; convert to JSON",
		}
	}
	if err := validateJSON(src); err != nil {
		return nil, &ParseError{
			Path:    path,
			Range:   base.ResolveRange(0, len(src)),
			Message: fmt.Sprintf("invalid JSON: %s", err.Error()),
			Cause:   err,
		}
	}

	p := &Plugin{Base: base}
	p.Name, p.NameRange = stringField(src, &base, "name")
	p.Version, p.VersionRange = stringField(src, &base, "version")
	p.Description, _ = stringField(src, &base, "description")
	p.Commands = stringArrayField(src, "commands")
	p.Skills = stringArrayField(src, "skills")
	p.Agents = stringArrayField(src, "agents")
	return p, nil
}

// collectHooks walks the "hooks" stanza and flattens every
// (event, matcher, command) triple into Hook.Entries. The same nested
// shape applies to .claude/settings{,.local}.json, plugin
// hooks/hooks.json, and any .claude/hooks/*.json discovered by the
// classifier.
//
// The first return value reports whether the top-level "hooks" key was
// absent. Callers decide whether absence is an error (dedicated hook
// files) or acceptable (settings files that carry no hooks).
func collectHooks(src []byte, base *Base, h *Hook) (missing bool, err error) {
	hooksRaw, dt, _, gerr := jsonparser.Get(src, "hooks")
	if gerr != nil {
		if errors.Is(gerr, jsonparser.KeyPathNotFoundError) {
			return true, nil
		}
		return false, fmt.Errorf("hooks: %w", gerr)
	}
	if dt != jsonparser.Object {
		return false, fmt.Errorf("hooks must be an object, got %s", dt.String())
	}

	walkErr := jsonparser.ObjectEach(hooksRaw, func(eventKey, groups []byte, _ jsonparser.ValueType, _ int) error {
		event := string(eventKey)
		_, aerr := jsonparser.ArrayEach(groups, func(group []byte, _ jsonparser.ValueType, _ int, _ error) {
			collectMatcherGroup(group, base, event, h)
		})
		return aerr
	})
	return false, walkErr
}

// collectMatcherGroup pulls every { matcher, hooks: [...] } item out
// of one event's array and emits a HookEntry for each inner command.
//
// ArrayEach's offset parameter is the item's position within the
// enclosing buffer; ranges derived from it therefore refer to the
// group slice rather than the original file. v1 accepts that trade —
// rules point diagnostics at something meaningful even if the line
// numbers are relative to the matcher group's opening brace.
func collectMatcherGroup(group []byte, base *Base, event string, h *Hook) {
	matcher, matcherErr := jsonparser.GetString(group, "matcher")
	if matcherErr != nil {
		matcher = ""
	}
	_, aerr := jsonparser.ArrayEach(group, func(item []byte, _ jsonparser.ValueType, _ int, _ error) {
		entry := HookEntry{
			Event:   event,
			Matcher: matcher,
		}
		entry.Command, entry.CommandRange = stringField(item, base, "command")
		entry.Timeout, entry.TimeoutRange = intField(item, base, "timeout")
		h.Entries = append(h.Entries, entry)
	}, "hooks")
	if aerr != nil && !errors.Is(aerr, jsonparser.KeyPathNotFoundError) {
		// A malformed inner array is surfaced to callers as an empty
		// matcher group rather than a hard error so one bad entry in
		// a settings file does not mask the other well-formed hooks.
		return
	}
}

// stringField returns (value, Range) for a top-level string key. A
// missing or non-string key returns ("", zero Range).
func stringField(src []byte, base *Base, key string) (string, diag.Range) {
	value, dt, off, err := jsonparser.Get(src, key)
	if err != nil || dt != jsonparser.String {
		return "", diag.Range{}
	}
	// value is the unquoted content; off points at the opening quote.
	end := min(off+2+len(value), len(src))
	return string(value), base.ResolveRange(off, end)
}

// intField returns (value, Range) for a top-level integer key. A
// missing, float, or non-number key returns (0, zero Range).
func intField(src []byte, base *Base, key string) (int, diag.Range) {
	value, dt, off, err := jsonparser.Get(src, key)
	if err != nil || dt != jsonparser.Number {
		return 0, diag.Range{}
	}
	n, perr := strconv.Atoi(string(value))
	if perr != nil {
		return 0, diag.Range{}
	}
	end := off + len(value)
	return n, base.ResolveRange(off, end)
}

// stringArrayField returns all string entries of an array key.
// Missing or non-array keys return nil.
func stringArrayField(src []byte, key string) []string {
	raw, dt, _, err := jsonparser.Get(src, key)
	if err != nil || dt != jsonparser.Array {
		return nil
	}
	var out []string
	_, aerr := jsonparser.ArrayEach(raw, func(value []byte, vdt jsonparser.ValueType, _ int, _ error) {
		if vdt == jsonparser.String {
			out = append(out, string(value))
		}
	})
	if aerr != nil {
		return nil
	}
	return out
}

// validateJSON catches syntactic problems jsonparser would silently
// tolerate. It is a thin wrapper around jsonparser.Get with an empty
// path; if the doc parses as an object or array we accept it.
func validateJSON(src []byte) error {
	trimmed := bytes.TrimSpace(src)
	if len(trimmed) == 0 {
		return fmt.Errorf("empty file")
	}
	switch trimmed[0] {
	case '{', '[':
		// Ask jsonparser to walk the top-level value. Any syntactic
		// issue bubbles up.
		_, _, _, err := jsonparser.Get(src)
		if err != nil && !errors.Is(err, jsonparser.KeyPathNotFoundError) {
			return err
		}
		return nil
	default:
		return fmt.Errorf("expected JSON object or array")
	}
}

// isJSONFile reports whether path ends with .json.
func isJSONFile(p string) bool {
	return bytes.HasSuffix([]byte(p), []byte(".json"))
}

// isSettingsFile reports whether path names a Claude Code settings
// file. It must be precise: a hook file that happens to be called
// "settings.json" outside .claude/ is still a flat hook object.
func isSettingsFile(p string) bool {
	return bytes.HasSuffix([]byte(p), []byte("/.claude/settings.json")) ||
		bytes.HasSuffix([]byte(p), []byte("/.claude/settings.local.json")) ||
		bytes.Equal([]byte(p), []byte(".claude/settings.json")) ||
		bytes.Equal([]byte(p), []byte(".claude/settings.local.json"))
}
