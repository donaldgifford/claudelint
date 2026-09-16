package upstream

import "regexp"

// Anchors on the skills documentation page.
var (
	anchorSkillsFrontmatter   = headingAnchor(3, "Frontmatter reference")
	anchorSkillsPortable      = headingAnchor(4, "Using skill frontmatter outside Claude Code")
	anchorSkillsSubstitutions = headingAnchor(4, "Available string substitutions")
)

// colPortableFields is the second column of the portability table,
// which spells out the fields each distribution path accepts.
const colPortableFields = "frontmatter fields you can use"

// skillsFrontmatter reads the SKILL.md frontmatter reference table.
type skillsFrontmatter struct{}

var _ Extractor = skillsFrontmatter{}

func (skillsFrontmatter) Section() string        { return "skills.frontmatter" }
func (skillsFrontmatter) Source() string         { return SrcDocsSkills }
func (skillsFrontmatter) Anchor() *regexp.Regexp { return anchorSkillsFrontmatter }

func (skillsFrontmatter) Extract(section string, out *Digest) error {
	t, err := firstTableWithHeaders(section, colField, colRequired)
	if err != nil {
		return err
	}

	out.Skills.Frontmatter = fieldRows(t, "", "")

	return nonEmpty(out.Skills.Frontmatter, "skill frontmatter fields")
}

// skillsPortableFields reads the subset of frontmatter usable outside
// Claude Code. The fields are named in the table's second column, not
// its first, so reading that column rather than every code span in the
// section keeps the surrounding prose's script names out of the set.
type skillsPortableFields struct{}

var _ Extractor = skillsPortableFields{}

func (skillsPortableFields) Section() string        { return "skills.portable_fields" }
func (skillsPortableFields) Source() string         { return SrcDocsSkills }
func (skillsPortableFields) Anchor() *regexp.Regexp { return anchorSkillsPortable }

func (skillsPortableFields) Extract(section string, out *Digest) error {
	t, err := firstTableWithHeaders(section, "distribution path", colPortableFields)
	if err != nil {
		return err
	}

	col := t.Column(colPortableFields)

	var fields []string
	for i := range t.Rows {
		fields = append(fields, BacktickedTokens(t.Cell(i, col))...)
	}

	out.Skills.PortableFields = fields

	return nonEmpty(fields, "portable skill frontmatter fields")
}

// skillsSubstitutions reads the string substitutions available in skill
// bodies.
type skillsSubstitutions struct{}

var _ Extractor = skillsSubstitutions{}

func (skillsSubstitutions) Section() string        { return "skills.substitutions" }
func (skillsSubstitutions) Source() string         { return SrcDocsSkills }
func (skillsSubstitutions) Anchor() *regexp.Regexp { return anchorSkillsSubstitutions }

func (skillsSubstitutions) Extract(section string, out *Digest) error {
	t, err := firstTableWithHeaders(section, "variable")
	if err != nil {
		return err
	}

	col := t.Column("variable")

	vars := make([]string, 0, len(t.Rows))
	for i := range t.Rows {
		if v := Name(t.Cell(i, col)); v != "" {
			vars = append(vars, v)
		}
	}

	out.Skills.Substitutions = vars

	return nonEmpty(vars, "skill string substitutions")
}
