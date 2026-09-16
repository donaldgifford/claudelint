package upstream

import (
	"encoding/json"
	"fmt"
	"io"
)

// encodeJSON writes v as canonical JSON: two-space indent, LF line
// endings, exactly one trailing newline, and no HTML escaping.
//
// Every committed artifact in this package goes through here so that
// regenerating it on an unchanged input is a no-op in git. Three
// encoding/json defaults would otherwise break that:
//
//   - json.Marshal escapes <, > and & to < and friends, with no
//     option to stop it. The portable name pattern and description
//     constraints contain those characters.
//   - json.MarshalIndent emits no trailing newline, so the file would
//     never match what an editor writes.
//   - Map keys are sorted but struct fields are emitted in declaration
//     order, which is why the digest types declare fields
//     alphabetically by tag.
func encodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	return nil
}
