package diag

// Diagnostic is the single unit of linter output. Rules produce
// Diagnostics; the engine sorts, dedupes, and filters them; reporters
// marshal them. Fields are ordered to match the output formats — Path
// and Range come before Message so golden JSON diffs read naturally.
type Diagnostic struct {
	RuleID   string   `json:"rule_id"`
	Severity Severity `json:"severity"`
	Path     string   `json:"path"`
	Range    Range    `json:"range"`
	Message  string   `json:"message"`
	Detail   string   `json:"detail,omitempty"`

	// Fix is defined but always nil in v1. It carries forward JSON
	// compatibility so that when `claudelint fix` ships in a later
	// release, existing JSON consumers do not need to change their
	// schema — the field just starts being populated. The omitempty tag
	// keeps it absent from v1 output.
	Fix *Fix `json:"fix,omitempty"`
}

// Fix is the forward-compatible placeholder for autofix data. v1 never
// populates it; the shape will be formalized when `claudelint fix`
// ships. It is kept as a named type (rather than an anonymous struct)
// so the JSON field name is already reserved and external consumers do
// not have to guess.
type Fix struct {
	Description string `json:"description,omitempty"`
	Edits       []Edit `json:"edits,omitempty"`
}

// Edit is a placeholder text edit for a future autofix. Left empty so
// v1 does not commit to a schema it cannot yet verify end-to-end.
type Edit struct {
	Range   Range  `json:"range"`
	NewText string `json:"new_text"`
}
