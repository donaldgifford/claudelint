package upstream

import (
	"regexp"
	"strings"
)

// Patterns over the two Python validators and the specification page.
// These sources are small, stable, and not Markdown tables, so each
// fact has its own literal pattern rather than a shared scanner.
var (
	// pyMaxConstant matches "MAX_DESCRIPTION_LENGTH = 1024".
	pyMaxConstant = regexp.MustCompile(`(?m)^(MAX_[A-Z0-9_]+)\s*=\s*(\d+)\s*$`)
	// pySetLiteral matches an ALLOWED_* set literal and captures its
	// body, in either the multi-line or the single-line form both
	// validators use.
	pySetLiteral = regexp.MustCompile(`(?s)ALLOWED_[A-Z_]+\s*=\s*\{(.*?)\}`)
	// pyQuoted matches a single- or double-quoted Python string.
	pyQuoted = regexp.MustCompile(`'([^']*)'|"([^"]*)"`)
	// pyNamePattern matches the anchored name regex the validator
	// applies to a skill name.
	pyNamePattern = regexp.MustCompile(`re\.match\(\s*r'(\^\[a-z0-9[^']*)'`)
	// pyDescriptionMessage matches a rejection message about the
	// description field.
	pyDescriptionMessage = regexp.MustCompile(`return False,\s*f?["']([Dd]escription[^"']*)["']`)
)

// maxConstantFields maps the validator's length constants to the
// frontmatter field each one bounds.
var maxConstantFields = map[string]string{
	"MAX_SKILL_NAME_LENGTH":    "name",
	"MAX_DESCRIPTION_LENGTH":   "description",
	"MAX_COMPATIBILITY_LENGTH": "compatibility",
}

// portableSpec reads the Agent Skills specification's frontmatter
// table, which is the portable field set a skill may use outside Claude
// Code.
type portableSpec struct{}

var _ Extractor = portableSpec{}

func (portableSpec) Section() string        { return "portable.allowed_fields" }
func (portableSpec) Source() string         { return SrcAgentSkillsSpec }
func (portableSpec) Anchor() *regexp.Regexp { return nil }

func (portableSpec) Extract(section string, out *Digest) error {
	t, err := firstTableWithHeaders(section, colField, colRequired)
	if err != nil {
		return err
	}

	col := t.Column(colField)

	fields := make([]string, 0, len(t.Rows))
	for i := range t.Rows {
		if name := Name(t.Cell(i, col)); name != "" {
			fields = append(fields, name)
		}
	}

	out.Portable.AllowedFields = fields

	return nonEmpty(fields, "portable allowed fields")
}

// portableValidator reads the reference validator's length limits.
type portableValidator struct{}

var _ Extractor = portableValidator{}

func (portableValidator) Section() string        { return "portable.limits" }
func (portableValidator) Source() string         { return SrcAgentSkillsValidator }
func (portableValidator) Anchor() *regexp.Regexp { return nil }

func (portableValidator) Extract(section string, out *Digest) error {
	limits := make(map[string]int)

	for _, m := range pyMaxConstant.FindAllStringSubmatch(section, -1) {
		limits[limitField(m[1])] = atoi([]byte(m[2]))
	}

	out.Portable.Limits = limits

	return nonEmptyMap(limits, "portable length limits")
}

// limitField maps a length constant to the field it bounds, falling
// back to the constant's middle segment so a newly added limit is still
// recorded rather than dropped.
func limitField(constant string) string {
	if field, ok := maxConstantFields[constant]; ok {
		return field
	}

	trimmed := strings.TrimSuffix(strings.TrimPrefix(constant, "MAX_"), "_LENGTH")

	return strings.ToLower(trimmed)
}

// portableAnthropic reads Anthropic's own strict skill validator: the
// allowlist it enforces, the name pattern, and what it rejects in a
// description.
type portableAnthropic struct{}

var _ Extractor = portableAnthropic{}

func (portableAnthropic) Section() string        { return "portable.anthropic_allowed_fields" }
func (portableAnthropic) Source() string         { return SrcAnthropicQuickValid }
func (portableAnthropic) Anchor() *regexp.Regexp { return nil }

func (portableAnthropic) Extract(section string, out *Digest) error {
	out.Portable.AnthropicAllowedFields = pythonSetMembers(section)

	if m := pyNamePattern.FindStringSubmatch(section); m != nil {
		out.Portable.NamePattern = m[1]
	}

	for _, m := range pyDescriptionMessage.FindAllStringSubmatch(section, -1) {
		out.Portable.DescriptionConstraints = append(out.Portable.DescriptionConstraints, m[1])
	}

	return nonEmpty(out.Portable.AnthropicAllowedFields, "anthropic allowed skill fields")
}

// pythonSetMembers returns the quoted members of the first ALLOWED_*
// set literal in src.
func pythonSetMembers(src string) []string {
	body := pySetLiteral.FindStringSubmatch(src)
	if body == nil {
		return nil
	}

	var out []string
	for _, m := range pyQuoted.FindAllStringSubmatch(body[1], -1) {
		value := m[1]
		if value == "" {
			value = m[2]
		}
		if value != "" {
			out = append(out, value)
		}
	}

	return out
}
