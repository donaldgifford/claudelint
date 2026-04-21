package artifact

// ParseClaudeMD parses a CLAUDE.md file into a typed *ClaudeMD.
// CLAUDE.md files usually have no frontmatter; when they do, keys are
// preserved with their byte-accurate ranges for rules that care.
func ParseClaudeMD(path string, src []byte) (*ClaudeMD, *ParseError) {
	doc, err := parseMarkdown(path, src)
	if err != nil {
		return nil, err
	}
	return &ClaudeMD{
		Base:        doc.Base,
		Frontmatter: doc.Frontmatter,
		Body:        doc.Body,
	}, nil
}

// ParseSkill parses a SKILL.md file into a typed *Skill. Companion
// file indexing is performed by IndexSkillCompanions, which the
// discovery/engine wiring calls after a successful parse.
func ParseSkill(path string, src []byte) (*Skill, *ParseError) {
	doc, err := parseMarkdown(path, src)
	if err != nil {
		return nil, err
	}
	s := &Skill{
		Base:        doc.Base,
		Frontmatter: doc.Frontmatter,
		Body:        doc.Body,
	}
	s.Name = doc.asString("name")
	s.Description = doc.asString("description")
	s.Model = doc.asString("model")
	s.AllowedTools = doc.asStringList("allowed-tools")
	return s, nil
}

// ParseCommand parses a slash-command .md file into a typed *Command.
func ParseCommand(path string, src []byte) (*Command, *ParseError) {
	doc, err := parseMarkdown(path, src)
	if err != nil {
		return nil, err
	}
	c := &Command{
		Base:        doc.Base,
		Frontmatter: doc.Frontmatter,
		Body:        doc.Body,
	}
	c.Description = doc.asString("description")
	c.ArgumentHint = doc.asString("argument-hint")
	c.AllowedTools = doc.asStringList("allowed-tools")
	return c, nil
}

// ParseAgent parses a subagent .md file into a typed *Agent.
func ParseAgent(path string, src []byte) (*Agent, *ParseError) {
	doc, err := parseMarkdown(path, src)
	if err != nil {
		return nil, err
	}
	a := &Agent{
		Base:        doc.Base,
		Frontmatter: doc.Frontmatter,
		Body:        doc.Body,
	}
	a.Name = doc.asString("name")
	a.Description = doc.asString("description")
	a.Tools = doc.asStringList("tools")
	return a, nil
}
