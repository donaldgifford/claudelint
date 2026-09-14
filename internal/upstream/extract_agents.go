package upstream

import (
	"fmt"
	"regexp"
)

// anchorAgentsFrontmatter scopes to the subagent file reference. The
// frontmatter table sits under a deeper heading inside this section, so
// the extractor takes the first table whose columns match rather than
// anchoring on the deeper heading, which has been renamed before.
var anchorAgentsFrontmatter = headingAnchor(3, "Write subagent files")

// enumFields are the frontmatter fields whose description cell spells
// out the accepted values. Reading enums only for named fields is what
// keeps the extraction explicit rather than heuristic.
var enumFields = []string{
	"color",
	"effort",
	"isolation",
	"memory",
	"model",
	"permissionMode",
}

// agentsFrontmatter reads the subagent frontmatter field table.
type agentsFrontmatter struct{}

var _ Extractor = agentsFrontmatter{}

func (agentsFrontmatter) Section() string        { return "agents.frontmatter" }
func (agentsFrontmatter) Source() string         { return SrcDocsAgents }
func (agentsFrontmatter) Anchor() *regexp.Regexp { return anchorAgentsFrontmatter }

func (agentsFrontmatter) Extract(section string, out *Digest) error {
	t, err := agentFrontmatterTable(section)
	if err != nil {
		return err
	}

	out.Agents.Frontmatter = fieldRows(t, "", "")

	return nonEmpty(out.Agents.Frontmatter, "subagent frontmatter fields")
}

// agentsEnums reads the accepted values out of the description cells of
// the enum-valued frontmatter fields.
//
// It takes every inline code span in the cell, as DESIGN-0006 section 3
// specifies. A cell that also gives an example, such as the model row's
// full model ID, therefore contributes that example too; recording it
// as a deliberate deviation is what acknowledged.json is for.
type agentsEnums struct{}

var _ Extractor = agentsEnums{}

func (agentsEnums) Section() string        { return "agents.enums" }
func (agentsEnums) Source() string         { return SrcDocsAgents }
func (agentsEnums) Anchor() *regexp.Regexp { return anchorAgentsFrontmatter }

func (agentsEnums) Extract(section string, out *Digest) error {
	t, err := agentFrontmatterTable(section)
	if err != nil {
		return err
	}

	nameCol := t.Column(colField)
	descCol := t.Column("description")
	if descCol < 0 {
		return ErrHeaders
	}

	enums := make(map[string][]string, len(enumFields))

	for i := range t.Rows {
		name := Name(t.Cell(i, nameCol))
		if !isEnumField(name) {
			continue
		}

		values := BacktickedTokens(t.Cell(i, descCol))
		for j, v := range values {
			values[j] = unquote(v)
		}
		if len(values) > 0 {
			enums[name] = values
		}
	}

	out.Agents.Enums = enums

	return missingEnums(enums)
}

func isEnumField(name string) bool {
	for _, f := range enumFields {
		if f == name {
			return true
		}
	}

	return false
}

// missingEnums reports the named enum fields whose description cell
// yielded no values, which means the row was renamed or its wording
// stopped spelling the values out.
func missingEnums(got map[string][]string) error {
	var missing []string
	for _, f := range enumFields {
		if _, ok := got[f]; !ok {
			missing = append(missing, f)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("%w: no values for subagent enum fields %v", ErrEmptySection, missing)
}

// agentFrontmatterTable finds the frontmatter table: the first one in
// the section whose columns are field, required, description.
func agentFrontmatterTable(section string) (Table, error) {
	return firstTableWithHeaders(section, colField, colRequired, "description")
}
